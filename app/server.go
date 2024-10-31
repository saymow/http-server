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

type AppRequest struct {
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

	conn, err := l.Accept()

	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	buffer := make([]byte, 1024)
	conn.Read(buffer)

	appRequest := AppRequest{headers: make(map[string]string)}
	parseRequest(&appRequest, buffer)

	if appRequest.path == "/" {
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	} else if strings.HasPrefix(appRequest.path, "/echo") {
		echoHandler(conn, &appRequest)
	} else if strings.HasPrefix(appRequest.path, "/user-agent") {
		userAgentHandler(conn, &appRequest)
	} else {
		conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
	}
}

func parseRequest(appRequest *AppRequest, requestBuffer []byte) {
	requestStr := string(requestBuffer)
	parts := strings.Split(requestStr, "\r\n")
	target := strings.Split(parts[0], " ")
	idx := 1

	appRequest.path = target[1]
	for ; parts[idx] != ""; idx++ {
		header := strings.Split(parts[idx], ": ")
		appRequest.headers[header[0]] = header[1]
	}
	appRequest.body = parts[idx+1]
}

func userAgentHandler(conn net.Conn, request *AppRequest) {
	message := request.headers["User-Agent"]

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(message))))
	conn.Write([]byte(message))
	conn.Close()
}

func echoHandler(conn net.Conn, request *AppRequest) {
	message := strings.Replace(request.path, "/echo/", "", 1)

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Content-Type: text/plain\r\n"))
	conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(message))))
	conn.Write([]byte(message))
	conn.Close()
}
