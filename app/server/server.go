package server

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type HTTPProtocol struct {
	version     string
	path        string
	method      string
	Headers     map[string]string
	RouteParams map[string]string
	Body        string
}

type HTTPStatusCode struct {
	Ok       int
	Created  int
	NotFound int
}

type HTTPResponse struct {
	conn       net.Conn
	statusCode int
	headers    map[string]string
	body       string
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

type ServerError struct {
	message string
}

const (
	OPEN_PLACEHOLDER_CHAR  = '['
	CLOSE_PLACEHOLDER_CHAR = ']'
	WILDCARD_CHAR          = '*'
)

var HttpStatus = HTTPStatusCode{
	Ok:       200,
	Created:  201,
	NotFound: 404,
}

func Create() Router {
	return Router{}
}

func (error ServerError) Error() string {
	return fmt.Sprintf("Server error: %s", error.message)
}

func resolveTCPConnection(conn net.Conn) (*HTTPProtocol, error) {
	buffer := make([]byte, 1024)

	n, err := conn.Read(buffer)
	if err != nil {
		return nil, ServerError{"unexpected error."}
	}
	buffer = buffer[:n]

	request := string(buffer)
	parts := strings.Split(request, "\r\n")
	if len(parts) == 0 {
		return nil, ServerError{"malformated request."}
	}

	target := strings.Split(parts[0], " ")
	if len(target) != 3 {
		return nil, ServerError{"malformated request."}
	}

	protocol := HTTPProtocol{
		Headers: make(map[string]string),
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
			return nil, ServerError{"malformated request."}
		}

		protocol.Headers[header[0]] = header[1]
	}

	// Read possible body
	if idx+1 < len(parts) {
		protocol.Body = parts[idx+1]
	} else {
		protocol.Body = ""
	}

	return &protocol, nil
}

func (router *Router) Get(path string, handler RouteHandler) {
	router.getRoutes = append(router.getRoutes, Route{path, handler})
}

func getPathSegments(path string) []string {
	segments := []string{}

	for _, part := range strings.Split(path, "/") {
		if part != "" {
			segments = append(segments, part)
		}
	}

	return segments
}

func (router *Router) routeHandler(conn net.Conn, protocol *HTTPProtocol) error {
	for _, route := range router.getRoutes {
		if pathMatch(protocol.path, route.path) {
			protocol.RouteParams = getRouteParams(protocol.path, route.path)
			route.handler(protocol, &HTTPResponse{conn: conn, headers: make(map[string]string)})
			return nil
		}
	}

	return nil
}

func isPlaceholder(segment string) bool {
	return len(segment) > 2 &&
		segment[0] == OPEN_PLACEHOLDER_CHAR &&
		segment[len(segment)-1] == CLOSE_PLACEHOLDER_CHAR
}

func pathMatch(requestPath, routePath string) bool {
	requestSegments := getPathSegments(requestPath)
	routeSegments := getPathSegments(routePath)
	idx := 0

	for ; idx < len(requestSegments) && idx < len(routeSegments); idx++ {
		if routeSegments[idx] == string(WILDCARD_CHAR) {
			return true
		} else if !isPlaceholder(routeSegments[idx]) && requestSegments[idx] != routeSegments[idx] {
			return false
		}
	}

	return len(requestSegments) == len(routeSegments)
}

func stripPlaceholderChars(placeholder string) string {
	r, _ := regexp.Compile(`\[(.+)\]`)

	match := r.FindStringSubmatch(placeholder)

	return match[1]
}

func getRouteParams(requestPath, routePath string) map[string]string {
	routeParams := make(map[string]string)
	requestSegments := getPathSegments(requestPath)
	routeSegments := getPathSegments(routePath)

	if len(requestSegments) != len(routeSegments) {
		return make(map[string]string)
	}

	for idx := 0; idx < len(requestSegments); idx++ {
		if isPlaceholder(routeSegments[idx]) {
			routeParams[stripPlaceholderChars(routeSegments[idx])] = requestSegments[idx]
		} else if routeSegments[idx] != requestSegments[idx] {
			return make(map[string]string)
		}
	}

	return routeParams
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

func (response *HTTPResponse) StatusCode(statusCode int) (*HTTPResponse, error) {
	if response.sent {
		return nil, ServerError{"connection already closed."}
	}

	response.statusCode = statusCode
	return response, nil
}

func statusCodeLine(statusCode int) string {
	switch statusCode {
	case HttpStatus.Ok:
		return "HTTP/1.1 200 Ok\r\n"
	case HttpStatus.Created:
		return "HTTP/1.1 201 Created\r\n"
	case HttpStatus.NotFound:
		return "HTTP/1.1 404 Not Found\r\n"
	default:
		return "HTTP/1.1 200 Ok\r\n"
	}
}

func (response *HTTPResponse) Body(body string) (*HTTPResponse, error) {
	if response.sent {
		return nil, ServerError{"connection already closed."}
	}

	response.body = body
	return response, nil
}

func (response *HTTPResponse) SetHeader(key, value string) error {
	if response.sent {
		return ServerError{"connection already closed."}
	}

	response.headers[key] = value
	return nil
}

func (response *HTTPResponse) Send() error {
	if response.sent {
		return ServerError{"connection already closed."}
	}

	if _, err := response.conn.Write([]byte(statusCodeLine(response.statusCode))); err != nil {
		return err
	}

	for key, value := range response.headers {
		if _, err := response.conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
			return err
		}
	}

	if _, err := response.conn.Write([]byte("\r\n")); err != nil {
		return err
	}
	if _, err := response.conn.Write([]byte(response.body)); err != nil {
		return err
	}

	response.sent = true
	response.conn.Close()
	return nil
}
