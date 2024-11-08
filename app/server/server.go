package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

type HTTPProtocol struct {
	version     string
	method      string
	Path        string
	Headers     map[string][]string
	RouteParams map[string]string
	Body        string
}

type HTTPStatusCode struct {
	Ok                 int
	Created            int
	NotFound           int
	InternalSeverError int
}

type HTTPResponse struct {
	conn          net.Conn
	statusCode    int
	customHeaders map[string]string
	body          string
	headerSent    bool
	sent          bool
}

type RouteHandler func(protocol *HTTPProtocol, response *HTTPResponse)

type Route struct {
	path    string
	handler RouteHandler
}

type Router struct {
	getRoutes  []Route
	postRoutes []Route
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
	Ok:                 200,
	Created:            201,
	NotFound:           404,
	InternalSeverError: 500,
}

func Create() Router {
	return Router{}
}

func (error ServerError) Error() string {
	return fmt.Sprintf("Server error: %s", error.message)
}

func (router *Router) Get(path string, handler RouteHandler) {
	router.getRoutes = append(router.getRoutes, Route{path, handler})
}

func (router *Router) Post(path string, handler RouteHandler) {
	router.postRoutes = append(router.postRoutes, Route{path, handler})
}

func resolveConnection(conn net.Conn) (*HTTPProtocol, error) {
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
		Headers: make(map[string][]string),
	}

	// Read HTTP target
	protocol.method = target[0]
	protocol.Path = target[1]
	protocol.version = target[2]

	// Read HTTP headers
	idx := 1
	for ; idx < len(parts) && parts[idx] != ""; idx++ {
		header := strings.Split(parts[idx], ": ")

		if len(header) != 2 {
			return nil, ServerError{"malformated request."}
		}

		protocol.Headers[header[0]] = strings.Split(header[1], ", ")

	}

	// Read possible body
	if idx+1 < len(parts) {
		protocol.Body = parts[idx+1]
	} else {
		protocol.Body = ""
	}

	return &protocol, nil
}

func (router *Router) Listen(address string) error {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return err
	}

	for {
		conn, _ := listener.Accept()
		go router.connectionHandler(conn)
	}
}

func (router *Router) connectionHandler(conn net.Conn) error {
	protocol, err := resolveConnection(conn)

	if err != nil {
		conn.Close()
		return err
	}

	response := &HTTPResponse{conn: conn, customHeaders: make(map[string]string)}

	defer response.Close()

	if _, ok := protocol.Headers["Accept-Encoding"]; ok {
		if slices.Contains(protocol.Headers["Accept-Encoding"], "gzip") {
			response.SetHeader("Content-Encoding", "gzip")
		}
	}

	if protocol.method == "GET" {
		for _, route := range router.getRoutes {
			if pathMatch(protocol.Path, route.path) {
				protocol.RouteParams = getRouteParams(protocol.Path, route.path)
				route.handler(protocol, response)
				return nil
			}
		}
	} else if protocol.method == "POST" {
		for _, route := range router.postRoutes {
			if pathMatch(protocol.Path, route.path) {
				protocol.RouteParams = getRouteParams(protocol.Path, route.path)
				route.handler(protocol, response)
				return nil
			}
		}
	}

	response.StatusCode(404)
	return nil
}

func isPlaceholder(segment string) bool {
	return len(segment) > 2 &&
		segment[0] == OPEN_PLACEHOLDER_CHAR &&
		segment[len(segment)-1] == CLOSE_PLACEHOLDER_CHAR
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

func (response *HTTPResponse) SetHeader(key, value string) error {
	if response.sent {
		return ServerError{"connection already closed."}
	}

	response.customHeaders[key] = value
	return nil
}

func (response *HTTPResponse) Body(body string) (*HTTPResponse, error) {
	if response.sent {
		return nil, ServerError{"connection already closed."}
	}

	response.body = body
	return response, nil
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
	case HttpStatus.InternalSeverError:
		return "HTTP/1.1 500 Internal Server Error\r\n"
	default:
		return "HTTP/1.1 200 OK\r\n"
	}
}

func (response *HTTPResponse) writeHeader(serverHeaders map[string]string) error {
	if response.headerSent {
		return ServerError{"header already sent."}
	}

	if _, err := response.conn.Write([]byte(statusCodeLine(response.statusCode))); err != nil {
		return err
	}

	for key, value := range serverHeaders {
		if _, err := response.conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
			return err
		}
	}

	for key, value := range response.customHeaders {
		if _, err := response.conn.Write([]byte(fmt.Sprintf("%s: %s\r\n", key, value))); err != nil {
			return err
		}
	}

	if _, err := response.conn.Write([]byte("\r\n")); err != nil {
		return err
	}

	response.headerSent = true
	return nil
}

func (response *HTTPResponse) Write(b []byte) (int, error) {
	if response.sent {
		return 0, ServerError{"connection already closed."}
	}

	if !response.headerSent {
		if err := response.writeHeader(map[string]string{}); err != nil {
			return 0, err
		}
	}

	n, err := response.conn.Write(b)

	if err != nil {
		return n, err
	}

	return n, nil
}

func (response *HTTPResponse) Send() error {
	if response.sent {
		return ServerError{"connection already closed."}
	}

	defer response.Close()

	if response.body == "" {
		return nil
	}

	serverHeaders := map[string]string{}
	var message []byte
	var messageLength int

	if response.customHeaders["Content-Encoding"] == "gzip" {
		var buffer bytes.Buffer
		gzipWriter := gzip.NewWriter(&buffer)

		if _, err := gzipWriter.Write([]byte(response.body)); err != nil {
			return err
		}

		if err := gzipWriter.Close(); err != nil {
			return err
		}

		message = buffer.Bytes()
		messageLength = buffer.Len()
	} else {
		message = []byte(response.body)
		messageLength = len(response.body)
	}

	serverHeaders["Content-Type"] = "plain/text"
	serverHeaders["Content-Length"] = strconv.Itoa(messageLength)

	if err := response.writeHeader(serverHeaders); err != nil {
		return err
	}
	if _, err := response.conn.Write(message); err != nil {
		return err
	}

	return nil
}

func (response *HTTPResponse) Close() error {
	if response.sent {
		return ServerError{"connection already closed."}
	}

	if !response.headerSent {
		if err := response.writeHeader(map[string]string{}); err != nil {
			return err
		}
	}

	response.conn.Close()
	response.sent = true
	return nil
}
