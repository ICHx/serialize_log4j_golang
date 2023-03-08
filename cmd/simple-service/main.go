package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/caarlos0/env/v7"
	"github.com/jkeys089/jserial"
)

type config struct {
	HOST string `env:"HOST_LOG4J_INPUT" envDefault:"localhost"`
	PORT string `env:"PORT_LOG4J_INPUT" envDefault:"2518"`
	// BUFFER int    `env:"BUFFER_LOG4J_INPUT" envDefault:"1024"`
}

func Listen(address string) net.Listener {
	log.Default().Println("Listening on tcp", address)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Error listening:", err.Error())
		os.Exit(1)
	}
	return listen
}

var cfg config

func main() {
	// get the addr from the environment
	if err := env.Parse(&cfg); err != nil {
		log.Println("Error parsing env. ", err)
	}
	// Listen for incoming connections.
	l := Listen(cfg.HOST + ":" + cfg.PORT)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Panicln("Error accepting: ", err.Error())
			continue
		}
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()
	// Make a reader to get incoming data.

	objs := read_stream(conn)
	log.Println("Received unserialized content:\n", objs)
}

func Uint16(b []byte) uint16 {
	// big endian
	result := uint16(b[1]) | uint16(b[0])<<8
	log.Println("Uint16:", result)
	return result
}

// read from start, until magic footer is found
func read_until_footer(r *bufio.Reader) ([]byte, error) {
	magic_header := []byte{0xac, 0xed, 0x00, 0x05}
	var buf bytes.Buffer

	is_object_footer := false
	for {
		b, err := r.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error peeking:", err)
			break
		}
		if b[0] == 0x70 {
			if err != nil {
				log.Println("Error peeking:", err)
				break
			}
		} else {
			continue
		}

		b, err = r.Peek(3)
		if err != nil {
			log.Println("Error peeking 3bytes:", err)
		}
		// read 3 bytes
		if b[1] == 0x78 && b[2] == 0x79 {
			r.Discard(3)
			is_object_footer = true
		}

		if is_object_footer {
			buf.Write(magic_header)
			// obj_stream:= r.Sc()
			// buf.Write
		}
	}
	return buf.Bytes(), nil
}

func read_stream(conn net.Conn) string {

	r := bufio.NewReader(conn)

	// split the stream into object_streams
	// read until encountering 70 78 79
	// then split the stream into object_streams
	var java_object_streams [][]byte

	// while not EOF, write splitted streams to java_object_streams
	for {
		obj_bytes, _ := read_until_footer(r)

		java_object_streams = append(java_object_streams, obj_bytes)
	}

	var deserialized_objects []interface{}

	for _, buf := range java_object_streams {
		obj_arr, err := jserial.ParseSerializedObjectMinimal(buf)
		// workaround: ALWAYS being parsed as list of one object
		if err != nil {
			if err == io.EOF && obj_arr == nil {
				continue
			}
			if strings.Contains(err.Error(), "parsing Reset") {
				// ignore error
			} else {
				log.Println("Error parsing object:", obj_arr, err)
				continue
			}
		}
		obj := obj_arr[0]
		deserialized_objects = append(deserialized_objects, obj)
	}

	var json_str_array []string
	for _, obj := range deserialized_objects {
		json_str_array = append(json_str_array, to_json(obj))
	}

	return strings.Join(json_str_array, ",")
}

func to_json(t interface{}) string {
	json_obj, err := json.Marshal(t)
	if err != nil {
		log.Println("Error marshalling object:", err)
		return ""
	}
	return string(json_obj)
}
