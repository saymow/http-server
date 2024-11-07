package server

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
)

type HTTPClientResponse struct {
	Version        string
	StatusCode     int
	StatusCodeText string
	Headers        map[string]string
	Body           string
}

func readConnectionResponse(conn net.Conn) (string, error) {
	var stringBuilder strings.Builder
	buffer := make([]byte, 1024)

	for {
		n, err := conn.Read(buffer)

		if err != nil && err != io.EOF {
			return stringBuilder.String(), err
		}
		if n == 0 {
			break
		}

		stringBuilder.Write(buffer[:n])
	}

	return stringBuilder.String(), nil
}

func readHTTPResponse(conn net.Conn) (*HTTPClientResponse, error) {
	connResponse, err := readConnectionResponse(conn)

	if err != nil {
		return nil, err
	}

	httpResponse := HTTPClientResponse{Headers: make(map[string]string)}

	parts := strings.Split(connResponse, "\r\n")

	if len(parts) == 0 {
		return nil, fmt.Errorf("Invalid http response format")
	}

	target := strings.Split(parts[0], " ")

	if len(target) != 3 {
		return nil, fmt.Errorf("Invalid http response format")
	}

	statusCode, err := strconv.Atoi(target[1])

	if err != nil {
		return nil, fmt.Errorf("Invalid http response format")
	}

	// Read HTTP target
	httpResponse.Version = target[0]
	httpResponse.StatusCode = statusCode
	httpResponse.StatusCodeText = target[2]

	// Read HTTP headers
	idx := 1
	for ; idx < len(parts) && parts[idx] != ""; idx++ {
		header := strings.Split(parts[idx], ": ")

		if len(header) != 2 {
			return nil, fmt.Errorf("Invalid http response format")
		}

		httpResponse.Headers[header[0]] = header[1]
	}

	if idx+1 < len(parts) {
		httpResponse.Body = parts[idx+1]
	}

	return &httpResponse, nil
}

func TestBasicConnection(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer func() {
		client.Close()
		server.Close()
	}()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Send()
	})

	go (func() {
		client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	})()

	go (func() {
		router.connectionHandler(server)
	})()

	response, _ := readConnectionResponse(client)
	Assert(t, strconv.Quote(response), strconv.Quote("HTTP/1.1 200 OK\r\n\r\n"))
}

func TestResponseFormat(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer func() {
		client.Close()
		server.Close()
	}()

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

	response, _ := readConnectionResponse(client)
	Assert(t, strconv.Quote(response), strconv.Quote("HTTP/1.1 200 OK\r\nContent-Type: plain/text\r\nContent-Length: 17\r\n\r\nthe response body"))
}

func TestRouteParams(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer func() {
		client.Close()
		server.Close()
	}()

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

	readConnectionResponse(client)
}

func TestCustomHeaders(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer func() {
		client.Close()
		server.Close()
	}()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.SetHeader("Set-Cookie", "key=value; HttpOnly")
		response.SetHeader("Cache-Control", "max-age=604800")
		response.Body("a rather expensive body")
		response.Send()
	})

	go (func() {
		client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	})()

	go (func() {
		router.connectionHandler(server)
	})()

	response, _ := readHTTPResponse(client)

	Assert(t, response.Version, "HTTP/1.1")
	Assert(t, response.StatusCode, 200)
	Assert(t, response.StatusCodeText, "OK")
	Assert(t, response.Headers["Content-Type"], "plain/text")
	Assert(t, response.Headers["Content-Length"], "23")
	Assert(t, response.Headers["Cache-Control"], "max-age=604800")
	Assert(t, response.Headers["Set-Cookie"], "key=value; HttpOnly")
	Assert(t, response.Body, "a rather expensive body")
}
