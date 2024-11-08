package server

import (
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

	r, _ := regexp.Compile(`(HTTP\/1\.1) (\d{3}) (.*)`)

	target := r.FindStringSubmatch(parts[0])

	if len(target) != 4 {
		return nil, fmt.Errorf("Invalid http target line")
	}

	statusCode, err := strconv.Atoi(target[2])

	if err != nil {
		return nil, fmt.Errorf("Invalid http status code")
	}

	// Read HTTP target
	httpResponse.Version = target[1]
	httpResponse.StatusCode = statusCode
	httpResponse.StatusCodeText = target[3]

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

	defer client.Close()
	defer server.Close()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Send()
	})

	go client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	go router.connectionHandler(server)

	response, _ := readConnectionResponse(client)
	assert.Equal(t, strconv.Quote(response), strconv.Quote("HTTP/1.1 200 OK\r\n\r\n"))
}

func TestResponseFormat(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer client.Close()
	defer server.Close()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Body("the response body")
		response.Send()
	})

	go client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	go router.connectionHandler(server)

	response, err := readHTTPResponse(client)

	assert.Nil(t, err)
	assert.Equal(t, response.Version, "HTTP/1.1")
	assert.Equal(t, response.StatusCode, 200)
	assert.Equal(t, response.StatusCodeText, "OK")
	assert.Equal(t, response.Headers["Content-Type"], "plain/text")
	assert.Equal(t, response.Headers["Content-Length"], "17")
	assert.Equal(t, response.Body, "the response body")
}

func TestRouteParams(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer client.Close()
	defer server.Close()

	router.Get("/users/[userId]/department/[userDepartment]", func(protocol *HTTPProtocol, response *HTTPResponse) {
		assert.Equal(t, protocol.RouteParams["userId"], "77")
		assert.Equal(t, protocol.RouteParams["userDepartment"], "accounting")
		response.Close()
	})

	go client.Write([]byte("GET /users/77/department/accounting HTTP/1.1\r\n\r\n"))
	go router.connectionHandler(server)

	readConnectionResponse(client)
}

func TestCustomHeaders(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer client.Close()
	defer server.Close()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.SetHeader("Set-Cookie", "key=value; HttpOnly")
		response.SetHeader("Cache-Control", "max-age=604800")
		response.Body("a rather expensive body")
		response.Send()
	})

	go client.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	go router.connectionHandler(server)

	response, err := readHTTPResponse(client)

	assert.Nil(t, err)
	assert.Equal(t, response.Version, "HTTP/1.1")
	assert.Equal(t, response.StatusCode, 200)
	assert.Equal(t, response.StatusCodeText, "OK")
	assert.Equal(t, response.Headers["Content-Type"], "plain/text")
	assert.Equal(t, response.Headers["Content-Length"], "23")
	assert.Equal(t, response.Headers["Cache-Control"], "max-age=604800")
	assert.Equal(t, response.Headers["Set-Cookie"], "key=value; HttpOnly")
	assert.Equal(t, response.Body, "a rather expensive body")
}

func TestCatchAllRoutes(t *testing.T) {
	client, server := net.Pipe()
	router := Create()

	defer client.Close()
	defer server.Close()

	router.Get("/", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.Body("body message")
		response.Send()
	})

	router.Get("/*", func(protocol *HTTPProtocol, response *HTTPResponse) {
		response.StatusCode(HttpStatus.NotFound)
		response.Body(fmt.Sprintf("%s not found.", protocol.Path))
		response.Send()
	})

	go client.Write([]byte("GET /resource/6/details HTTP/1.1\r\n\r\n"))
	go router.connectionHandler(server)

	response, err := readHTTPResponse(client)

	assert.Nil(t, err)
	assert.Equal(t, response.Version, "HTTP/1.1")
	assert.Equal(t, response.StatusCode, 404)
	assert.Equal(t, response.StatusCodeText, "Not Found")
	assert.Equal(t, response.Body, "/resource/6/details not found.")
}
