# websocket
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/exelban/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/exelban/websocket?style=flat-square)](https://goreportcard.com/report/github.com/exelban/websocket)
[![Codecov](https://img.shields.io/codecov/c/github/exelban/websocket.svg?style=flat-square)](https://codecov.io/gh/exelban/websocket) 

Simple websocket library for golang

# Feature Overview
- onConnect/onDisconnect handlers
- message routing (similary to implementation in socket.io)
- groups (channels) implementation
- using net.Conn as connection
- lightweight and easy to use

# Example
## [Echo](https://github.com/exelban/websocket/blob/channels/example/echo.go)
```golang
package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	ws := websocket.CreateAndRun()

	r.Get("/ws", ws.Handler)
	ws.On("echo", func(c *websocket.Conn, msg *websocket.Message) {
		c.Emit("echo", msg.Body)
	})

	http.ListenAndServe(":8080", r)
}
```

## [Channel](https://github.com/exelban/websocket/blob/channels/example/channel.go)
```golang
package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	ws := websocket.CreateAndRun()

	ch := ws.NewChannel("test")

	ws.OnConnect(func(c *websocket.Conn) {
		ch.Add(c)
		ch.Emit("connection", []byte("new connection come"))
	})

	r.Get("/ws", ws.Handler)
	http.ListenAndServe(":8080", r)
}
```

# Benchmark

# Licence
[MIT License](https://github.com/exelban/websocket/blob/channels/LICENSE)
