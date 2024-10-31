package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "net" and "os" imports above (feel free to remove this!)
var _ = net.Listen
var _ = os.Exit

type HTTPProtocol struct {
	path    string
	headers map[string]string
	body    string
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:4221")

	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go requestHanlder(conn)
	}
}

func requestHanlder(conn net.Conn) {
	buffer := make([]byte, 1024)
	conn.Read(buffer)

	protocol := HTTPProtocol{
		headers: make(map[string]string),
	}
	parseProtocol(&protocol, buffer)

	if protocol.path == "/" {
		indexHandler(conn, &protocol)
	} else if strings.HasPrefix(protocol.path, "/echo") {
		echoHandler(conn, &protocol)
	} else if strings.HasPrefix(protocol.path, "/user-agent") {
		userAgentHandler(conn, &protocol)
	} else {
		notFoundHandler(conn, &protocol)
	}
}

func parseProtocol(protocol *HTTPProtocol, requestBuffer []byte) {
	requestStr := string(requestBuffer)
	parts := strings.Split(requestStr, "\r\n")
	target := strings.Split(parts[0], " ")
	idx := 1

	protocol.path = target[1]
	for ; parts[idx] != ""; idx++ {
		header := strings.Split(parts[idx], ": ")
		protocol.headers[header[0]] = header[1]
	}

	if idx+1 < len(parts) {
		protocol.body = parts[idx+1]
	} else {
		protocol.body = ""
	}
}

func indexHandler(conn net.Conn, request *HTTPProtocol) {
	conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	conn.Close()
}

func userAgentHandler(conn net.Conn, request *HTTPProtocol) {
	message := request.headers["User-Agent"]

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(message))))
	conn.Write([]byte(message))
	conn.Close()
}

func echoHandler(conn net.Conn, request *HTTPProtocol) {
	message := strings.Replace(request.path, "/echo/", "", 1)

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(message))))
	conn.Write([]byte(message))
	conn.Close()
}

func notFoundHandler(conn net.Conn, request *HTTPProtocol) {
	conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	conn.Close()
}
