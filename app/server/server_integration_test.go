package server

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

func readResponse(conn net.Conn) (string, error) {
	var stringBuilder strings.Builder
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)

		if err != nil {
			return stringBuilder.String(), err
		}
		if n == 0 {
			break
		}

		stringBuilder.Write(buffer[:n])
	}

	return stringBuilder.String(), nil
}

func TestConnection(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Send()
	})

	go (func() {
		client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	})()

	go (func() {
		router.connectionHandler(server)
	})()

	response, _ := readResponse(client)
	Assert(t, strconv.Quote(response), strconv.Quote("HTTP/1.1 200 OK\r\n\r\n"))
}
