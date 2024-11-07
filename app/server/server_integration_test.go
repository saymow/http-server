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

func TestBasicConnection(t *testing.T) {
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

func TestResponseBody(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Body("the response body")
		response.Send()
	})

	go (func() {
		client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	})()

	go (func() {
		router.connectionHandler(server)
	})()

	response, _ := readResponse(client)
	Assert(t, strconv.Quote(response), strconv.Quote("HTTP/1.1 200 OK\r\nContent-Type: plain/text\r\nContent-Length: 17\r\n\r\nthe response body"))
}

func TestRouteParams(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	router.Get("/users/[userId]/department/[userDepartment]", func(protocol *HTTPProtocol, response *HTTPResponse) {
		Assert(t, protocol.RouteParams["userId"], "77")
		Assert(t, protocol.RouteParams["userDepartment"], "accounting")
		response.Close()
	})

	go (func() {
		client.Write([]byte("GET /users/77/department/accounting HTTP/1.1\r\n\r\n"))
	})()

	go (func() {
		router.connectionHandler(server)
	})()

	readResponse(client)
}
