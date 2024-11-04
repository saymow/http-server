package server

import (
	"net"
	"strings"
)

type HTTPProtocol struct {
	version string
	path    string
	method  string
	headers map[string]string
	body    string
}

type HTTPStatusCode struct {
	Ok      int
	Created int
}

type HTTPResponse struct {
	conn       net.Conn
	statusCode int
	sent       bool
}

type RouteHandler func(protocol *HTTPProtocol, response *HTTPResponse)

type Route struct {
	path    string
	handler RouteHandler
}

type Router struct {
	getRoutes []Route
}

type MalformatedRequestError struct {
}

var HttpStatus = HTTPStatusCode{
	Ok:      200,
	Created: 201,
}

func Create() Router {
	return Router{}
}

func (error MalformatedRequestError) Error() string {
	return "Invalid request format"
}

func resolveTCPConnection(conn net.Conn) (*HTTPProtocol, error) {
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		return nil, MalformatedRequestError{}
	}
	buffer = buffer[:n]

	request := string(buffer)
	parts := strings.Split(request, "\r\n")
	if len(parts) == 0 {
		return nil, MalformatedRequestError{}
	}

	target := strings.Split(parts[0], " ")
	if len(target) != 3 {
		return nil, MalformatedRequestError{}
	}

	protocol := HTTPProtocol{
		headers: make(map[string]string),
	}

	// Read HTTP target
	protocol.method = target[0]
	protocol.path = target[1]
	protocol.version = target[2]

	// Read HTTP headers
	idx := 1
	for ; idx < len(parts) && parts[idx] != ""; idx++ {
		header := strings.Split(parts[idx], ": ")

		if len(header) != 2 {
			return nil, MalformatedRequestError{}
		}

		protocol.headers[header[0]] = header[1]
	}

	// Read possible body
	if idx+1 < len(parts) {
		protocol.body = parts[idx+1]
	} else {
		protocol.body = ""
	}

	return &protocol, nil
}

func (router *Router) Get(path string, handler RouteHandler) {
	router.getRoutes = append(router.getRoutes, Route{path, handler})
}

func pathMatch(routePath, protocolPath string) bool {
	return true
}

func (router *Router) routeHandler(conn net.Conn, protocol *HTTPProtocol) error {
	for _, route := range router.getRoutes {
		if pathMatch(route.path, protocol.path) {
			route.handler(protocol, &HTTPResponse{conn: conn})
			return nil
		}
	}

	return nil
}

func (router *Router) Listen(address string) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return err
	}

	for {
		conn, _ := listener.Accept()
		httpProtocol, _ := resolveTCPConnection(conn)

		go router.routeHandler(conn, httpProtocol)
	}
}

func (response *HTTPResponse) StatusCode(statusCode int) *HTTPResponse {
	response.statusCode = statusCode
	return response
}

func StatusCodeLine(statusCode int) string {
	switch statusCode {
	case HttpStatus.Ok:
		return "HTTP/1.1 200 Ok\r\n"
	case HttpStatus.Created:
		return "HTTP/1.1 201 Created\r\n"
	default:
		return "HTTP/1.1 200 Ok\r\n"
	}
}

func (response *HTTPResponse) Send() error {
	if _, err := response.conn.Write([]byte(StatusCodeLine(response.statusCode))); err != nil {
		return err
	}
	if _, err := response.conn.Write([]byte("\r\n")); err != nil {
		return err
	}

	response.conn.Close()
	return nil
}
