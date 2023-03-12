package main

import (
	"bufio"
	"bytes"
	_ "encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/caarlos0/env/v7"
	"github.com/jkeys089/jserial"
)

type config struct {
	LISTEN_HOST string `env:"HOST_LOG4J_INPUT" envDefault:"localhost"`
	LISTEN_PORT string `env:"PORT_LOG4J_INPUT" envDefault:"2518"`
	// BUFFER int    `env:"BUFFER_LOG4J_INPUT" envDefault:"1024"`

	JSON_LOG_HOST string `env:"JSON_LOG_HOST" envDefault:"localhost"`
	JSON_LOG_PORT string `env:"JSON_LOG_PORT" envDefault:"5540"`
}

const (
	CONCURRENT_DESERIALIZE = 1000
)

func Listen(address string) net.Listener {
	log.Default().Println("Listening on tcp", address)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("Error listening:", err.Error())
	}
	return listen
}

func Dial(address string) net.Conn {
	log.Default().Println("Dialing to tcp", address)

	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", address)
		if err != nil {
			log.Println("Error dialing:", err.Error())
			time.Sleep(5 * time.Second)
		} else {
			return conn
		}
	}
	log.Panicln("Error dialing, stopped retrying")

	return nil
}

var cfg config

func main() {
	// get the addr from the environment
	if err := env.Parse(&cfg); err != nil {
		log.Println("Error parsing env. ", err)
	}
	// Listen for incoming connections.
	l := Listen(cfg.LISTEN_HOST + ":" + cfg.LISTEN_PORT)

	// create channel
	ch := make(chan string, CONCURRENT_DESERIALIZE)
	go handleFwd(ch)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting request: ", err.Error())
			continue
		}
		go handleRequest(conn, ch)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func handleFwd(data_ch chan string) {
	out_conn := Dial(cfg.JSON_LOG_HOST + ":" + cfg.JSON_LOG_PORT)
	log.Println("Handling Fwd", out_conn.RemoteAddr())
	defer out_conn.Close() // likely wouldn't close for entire program
	cnt := 0
	for data_obj := range data_ch {
		var show_len int = min((15), len(data_obj))
		object_excerpt := data_obj[:show_len]
		log.Print(fmt.Sprintf("Sending %d:", cnt), object_excerpt, "\t")

		for i := 0; i < 3; i++ {
			_, err := out_conn.Write([]byte(data_obj))
			if err != nil {
				log.Println("Error forwarding:", cnt, err.Error())
			} else {
				out_conn.Write([]byte("\n"))
				cnt++
				break
			}
		}
	}
}

func handleRequest(in_conn net.Conn, data_ch chan string) {
	log.Println("Handling Request", in_conn.RemoteAddr())
	defer log.Println("Finished Request", in_conn.RemoteAddr())
	defer in_conn.Close()
	// Make a reader to get incoming data.
	r := bufio.NewReader(in_conn)
	process_stream(r, data_ch)
}

func split_stream(sr *bufio.Reader, out_split_obj_ch chan []byte) {
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

	// first 2 bytes should be magic value 0xaced, second 2 bytes should be protocol version
	// hardcode to 0x0005
	if h, _ := sr.Peek(4); !bytes.Equal(h, stream_magic_header) {
		log.Println("Error reading magic header, got this instead: " + string(h))
	}
	sr.Discard(4)

	for { // finding objects in whole stream
		_, err := sr.Peek(1)
		if err == io.EOF {
			break
		}

		// is_object_footer := false
		has_err := false
		obj_write_buf := bytes.NewBuffer(make([]byte, 0, 1024))
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
		out_split_obj_ch <- obj_write_buf.Bytes()
	} // end of stream
	close(out_split_obj_ch)
}

func process_stream(cr *bufio.Reader, out_json_ch chan string) {
	// split the stream into object_streams
	// read until encountering 70 78 79
	// then split the stream into object_streams

	// while not EOF, write splitted streams to java_object_streams

	split_streams_ch := make(chan []byte, CONCURRENT_DESERIALIZE)
	go split_stream(cr, split_streams_ch)

	for stream := range split_streams_ch {
		obj_map, err := java_objstream_to_go_map(stream)
		if err != nil {
			log.Println("Error converting a java object to go object:", err)
			continue
		}

		obj_map = transform_log_event(obj_map)

		json_str, err := map_to_json(obj_map)
		if err != nil {
			log.Println("Error converting a go object to json:", err)
			continue
		}
		out_json_ch <- json_str
	}
}

func transform_log_event(obj_map map[string]interface{}) map[string]interface{} {
	var stack_trace_info, filename, methodname string
	if obj_map["throwableInfo"] != nil {
		stack_trace_info = strings.Join(obj_map["throwableStrRep"].([]string), "\n")
	}

	if obj_map["locationInformation"] != nil {
		filename = obj_map["locationInformation"].(map[string]interface{})["fileName"].(string)
		methodname = obj_map["locationInformation"].(map[string]interface{})["methodName"].(string)
	}

	// TODO: fix this
	level_byte := obj_map["@"].([]interface{})[0].([]byte)

	// 	for i,v:=range level_byte{
	// 	log.Println("level_bytes",i, v)
	// }
	log.Println("level_bytes", (level_byte))

	event := map[string]interface{}{
		"message":   obj_map["renderedMessage"],
		"timestamp": obj_map["timeStamp"],
		"path":      obj_map["loggerName"],
		// "priority": level_byte, //TODO hacking
		"logger_name": obj_map["categoryName"],
		"thread":      obj_map["threadName"],
		"class":       obj_map["categoryName"],
		"file":        filename,
		"method":      methodname,
		"ndc":         obj_map["ndc"],
		"stack_trace": stack_trace_info,
	}

	// event.set("message" => log4j_obj.getRenderedMessage)
	// event.set("timestamp", log4j_obj.getTimeStamp)
	// event.set("path", log4j_obj.getLoggerName)
	// event.set("priority", log4j_obj.getLevel.toString)
	// event.set("logger_name", log4j_obj.getLoggerName)
	// event.set("thread", log4j_obj.getThreadName)
	// event.set("class", log4j_obj.getLocationInformation.getClassName)
	// event.set("file", log4j_obj.getLocationInformation.getFileName + ":" + log4j_obj.getLocationInformation.getLineNumber)
	// event.set("method", log4j_obj.getLocationInformation.getMethodName)
	// event.set("NDC", log4j_obj.getNDC) if log4j_obj.getNDC
	// event.set("stack_trace", log4j_obj.getThrowableStrRep.to_a.join("\n")) if log4j_obj.getThrowableInformation

	return event
}

func map_to_json(t map[string]interface{}) (string, error) {
	json_obj, err := json.Marshal(t)
	if err != nil {
		log.Println("Error marshalling object:", err)
		return "", err
	}
	return string(json_obj), nil
}

func java_objstream_to_go_map(java_obj_bytes []byte) (map[string]interface{}, error) {
	obj_arr, err := jserial.ParseSerializedObject(java_obj_bytes)

	// obj_arr, err := jserial.ParseSerializedObjectMinimal(java_obj_bytes)
	if err != nil {
		if err == io.EOF && obj_arr == nil {
			return nil, err
		}
		if strings.Contains(err.Error(), "parsing Reset") {
			// ignore error
		} else {
			log.Println("Error parsing object:", obj_arr, err)
			log.Println("ascii:", string(java_obj_bytes))
			log.Println("=====================================")
			return nil, err
		}
	}
	// workaround: ALWAYS being parsed as list of one object
	var obj = obj_arr[0].(map[string]interface{})
	return obj, nil
}
