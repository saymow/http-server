package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/codecrafters-io/http-server-starter-go/app/server"
)

type HTTPProtocol struct {
	conn    net.Conn
	version string
	path    string
	method  string
	headers map[string]string
	body    string
}

func main() {
	router := server.Create()

	router.Get("/", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		response.StatusCode(server.HttpStatus.Created)
		response.Send()
	})

	router.Get("/echo/[message]", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		message := protocol.RouteParams["message"]

		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", strconv.Itoa(len(message)))
		response.Body(message)
		response.Send()
	})

	router.Get("/user-agent", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		userAgent := protocol.Headers["User-Agent"]

		response.SetHeader("Content-Type", "text/plain")
		response.SetHeader("Content-Length", strconv.Itoa(len(userAgent)))
		response.Body(userAgent)
		response.Send()
	})

	router.Get("*", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		response.StatusCode(server.HttpStatus.NotFound)
		response.Send()
	})

	router.Listen("0.0.0.0:4221")

	// l, err := net.Listen("tcp", "0.0.0.0:4221")

	// if err != nil {
	// 	fmt.Println("Failed to bind to port 4221")
	// 	os.Exit(1)
	// }

	// for {
	// 	conn, err := l.Accept()

	// 	if err != nil {
	// 		fmt.Println("Error accepting connection: ", err.Error())
	// 		os.Exit(1)
	// 	}

	// 	go requestHanlder(conn)
	// }
}

func requestHanlder(conn net.Conn) error {
	defer conn.Close()

	buffer := make([]byte, 1024)
	bytes_read, err := conn.Read(buffer)
	if err != nil {
		return err
	}
	buffer = buffer[:bytes_read]

	protocol := HTTPProtocol{
		headers: make(map[string]string),
		conn:    conn,
	}
	parseProtocol(&protocol, buffer)

	if strings.HasPrefix(protocol.path, "/files") && protocol.method == "GET" {
		getFileHandler(&protocol)
	} else if strings.HasPrefix(protocol.path, "/files") && protocol.method == "POST" {
		postFileHandler(&protocol)
	}

	return nil
}

func getFileHandler(request *HTTPProtocol) error {
	FILES_DIR := os.Args[2]
	filename := strings.Replace(request.path, "/files/", "", 1)
	filepath := FILES_DIR + filename

	file, err := os.Open(filepath)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			if _, err := request.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n")); err != nil {
				return err
			}
		}

		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	headers := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n", fileInfo.Size())

	if _, err := request.conn.Write([]byte(headers)); err != nil {
		return err
	}

	buffer := make([]byte, 1024)

	for {
		n, err := file.Read(buffer)

		if err != nil {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := request.conn.Write(buffer); err != nil {
			return err
		}
	}

	return nil
}

func postFileHandler(request *HTTPProtocol) error {
	FILES_DIR := os.Args[2]
	filename := strings.Replace(request.path, "/files/", "", 1)
	filepath := FILES_DIR + filename

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write([]byte(request.body)); err != nil {
		return err
	}

	if _, err := request.conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n")); err != nil {
		return err
	}

	return nil
}
