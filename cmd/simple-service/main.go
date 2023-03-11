package main

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"github.com/caarlos0/env/v7"
	"github.com/jkeys089/jserial"
)

type config struct {
	LISTEN_HOST string `env:"HOST_LOG4J_INPUT" envDefault:"localhost"`
	LISTEN_PORT string `env:"PORT_LOG4J_INPUT" envDefault:"2518"`
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
	l := Listen(cfg.LISTEN_HOST + ":" + cfg.LISTEN_PORT)
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
	log.Println("Total Deserialized messages:\n", len(objs))
}

// read from start, until magic footer is found
func split_stream(sr *bufio.Reader) ([][]byte, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in split_stream", r)
		}
	}()

	stream_magic_header := []byte{0xac, 0xed, 0x00, 0x05}
	object_footer := []byte{0x70, 0x78, 0x79}
	object_header := []byte{0x73, 0x72, 0x00, 0x21}
	// 0x73 0x72 0x00 is the header for object
	// 0x21 is the len of object FQCN "org.apache.log4j.spi.LoggingEvent"

	// inaccurate interpretation, but should work for now
	// actual specification mentioned in https://xz.aliyun.com/t/3847

	object_streams := [][]byte{}

	// first 2 bytes should be magic value 0xaced, second 2 bytes should be protocol version
	// hardcode to 0x0005
	if h, _ := sr.Peek(4); !bytes.Equal(h, stream_magic_header) {
		return nil,
			errors.New("Error reading magic header, got this instead: " + string(h))
	}
	sr.Discard(4)

	for { // finding objects in whole stream
		_, err := sr.Peek(1)
		if err == io.EOF {
			break
		}

		// is_object_footer := false
		has_err := false
		obj_write_buf := bytes.Buffer{}
		obj_write_buf.Write(stream_magic_header)

		for { // object: read bytes until magic footer is found
			byte_peek, err := sr.ReadByte()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Println("Error peeking:", err)
				has_err = true
				break
			}
			if byte_peek == 0x70 { // set is_object_footer
				byte_peek_2, err := sr.Peek(2)
				if err != nil {
					has_err = true
					if err == io.EOF {
						log.Panicln("Unexpected EOF while peeking for object footer")
					} else {
						log.Panicln("Error peeking for object footer:", err)
					}
				}
				// expect 0x78 0x79
				if bytes.Equal(byte_peek_2, object_footer[1:]) {
					obj_write_buf.Write(byte_peek_2)
					sr.Discard(2)

					// readahead to confirm next object header
					next_bytes, err := sr.Peek(4)
					if err == io.EOF || bytes.Equal(next_bytes, object_header[:]) {
						// is_object_footer = true
						break // end of current object
					} else {
						continue
					}
				} else {
					obj_write_buf.WriteByte(byte_peek)
					obj_write_buf.Write(byte_peek_2)
					sr.Discard(2)
					continue // continue reading
				}
			}

			if has_err {
				break
			}
			// not object footer, write 1 byte
			obj_write_buf.WriteByte(byte_peek)

		} // end of current object
		object_streams = append(object_streams, obj_write_buf.Bytes())
	} // end of stream
	return object_streams, nil
}

func read_stream(conn net.Conn) []string {

	// split the stream into object_streams
	// read until encountering 70 78 79
	// then split the stream into object_streams

	// while not EOF, write splitted streams to java_object_streams
	var java_object_streams [][]byte

	conn_reader := bufio.NewReader(conn)

	java_object_streams, err := split_stream(conn_reader)
	if err != nil {
		log.Println("Error splitting stream:", err)
	}
	log.Println("Splitted #", len(java_object_streams))

	var deserialized_objects []interface{}

	for i, stream := range java_object_streams {
		obj_arr, err := jserial.ParseSerializedObjectMinimal(stream)
		// workaround: ALWAYS being parsed as list of one object
		if err != nil {
			if err == io.EOF && obj_arr == nil {
				continue
			}
			if strings.Contains(err.Error(), "parsing Reset") {
				// ignore error
			} else {
				log.Println("Error parsing object:", i, obj_arr, err)
				log.Println("ascii:", string(stream))
				log.Println("hex:", hex.EncodeToString(stream))
				log.Println("=====================================")
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

	return json_str_array
}

func to_json(t interface{}) string {
	json_obj, err := json.Marshal(t)
	if err != nil {
		log.Println("Error marshalling object:", err)
		return ""
	}
	return string(json_obj)
}
