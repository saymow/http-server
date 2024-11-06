package main

import (
	"fmt"
	"os"

	"github.com/codecrafters-io/http-server-starter-go/app/server"
)

func main() {
	router := server.Create()

	router.Get("/", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		response.Send()
	})

	router.Get("/echo/[message]", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		message := protocol.RouteParams["message"]

		response.Body(message)
		response.Send()
	})

	router.Get("/user-agent", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		userAgent := protocol.Headers["User-Agent"][0]

		response.Body(userAgent)
		response.Send()
	})

	router.Get("/files/[filename]", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		FILES_DIR := os.Args[2]
		filename := protocol.RouteParams["filename"]
		filepath := FILES_DIR + filename

		file, err := os.Open(filepath)

		if err != nil {
			if _, ok := err.(*os.PathError); ok {
				response.StatusCode(server.HttpStatus.NotFound)
				response.Send()
				return
			}

			response.StatusCode(server.HttpStatus.InternalSeverError)
			response.Body(err.Error())
			response.Send()
			return
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			response.StatusCode(server.HttpStatus.InternalSeverError)
			response.Body(err.Error())
			response.Send()
			return
		}

		response.SetHeader("Content-Type", "application/octet-stream")
		response.SetHeader("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		buffer := make([]byte, 1024)

		for {
			n, err := file.Read(buffer)

			if err != nil {
				response.StatusCode(server.HttpStatus.InternalSeverError)
				response.Body(err.Error())
				response.Send()
				return
			} else if n == 0 {
				break
			}

			response.Write(buffer)
		}

		response.Close()
	})

	router.Get("*", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		response.StatusCode(server.HttpStatus.NotFound)
		response.Send()
	})

	router.Post("/files/[filename]", func(protocol *server.HTTPProtocol, response *server.HTTPResponse) {
		FILES_DIR := os.Args[2]
		filename := protocol.RouteParams["filename"]
		filepath := FILES_DIR + filename

		file, err := os.Create(filepath)
		if err != nil {
			response.StatusCode(server.HttpStatus.InternalSeverError)
			response.Body(err.Error())
			response.Send()
			return
		}
		defer file.Close()

		if _, err := file.Write([]byte(protocol.Body)); err != nil {
			response.StatusCode(server.HttpStatus.InternalSeverError)
			response.Body(err.Error())
			response.Send()
			return
		}

		response.StatusCode(server.HttpStatus.Created)
		response.Send()
	})

	router.Listen("0.0.0.0:4221")
}
