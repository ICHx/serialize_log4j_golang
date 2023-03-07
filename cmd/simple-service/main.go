package main

import (
	"bufio"
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

	objs := reader_to_json(conn)
	log.Println("Received unserialized content:\n", objs)
}

func reader_to_json(conn net.Conn) string {

	var json_str_array []string
	r := bufio.NewReader(conn)

	for {
		sop := jserial.NewSerializedObjectParser(r)
		if t, err := r.Peek(16); err != nil {
			log.Println("Error peeking reader:", err)
			log.Println("Peeked:", t)
			if err == io.EOF {
				if len(t) < 1 {
					break
				} else {
					r.ReadBytes(2) // read Uint16
				}
			}
		}

		obj_arr, err := sop.ParseSerializedObjectMinimal()
		if err != nil {
			if err == io.EOF && obj_arr == nil {
				continue
			}
			if strings.Contains(err.Error(), "parsing Reset") && obj_arr == nil {
				continue
			}
			log.Println("Error parsing object:", obj_arr, err)

		}

		for _, obj := range obj_arr {
			json_obj, err := json.Marshal(obj)
			json_str_array = append(json_str_array, string(json_obj))
			if err != nil {
				log.Println("Error marshalling object:", err)
				break
			}
		} // end for obj_arr
	} // end for sop parse

	return strings.Join(json_str_array, ",")
}
