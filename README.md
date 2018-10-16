# websocket
[![GoDoc](http://img.shields.io/badge/go-documentation-blue.svg?style=flat-square)](http://godoc.org/github.com/exelban/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/exelban/websocket)](https://goreportcard.com/report/github.com/exelban/websocket)
[![codecov](https://codecov.io/gh/exelban/websocket/branch/master/graph/badge.svg?token=A8eLVAj9cU)](https://codecov.io/gh/exelban/websocket)

Simple websocket library for golang

# Installation
```bash
go get github.com/exelban/websocket
```

# Example
## Echo
```golang
package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	wsServer := websocket.CreateAndRun()

	r.Get("/ws", wsServer.Handler)
	wsServer.On("echo", func(c *websocket.Conn, msg *websocket.Message) {
		c.Emit("echo", msg.Body)
	})

	http.ListenAndServe(":8080", r)
}
```

## Channel
```golang
package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	wsServer := websocket.CreateAndRun()

	ch := wsServer.NewChannel("test")

	wsServer.OnConnect(func(c *websocket.Conn) {
		ch.Add(c)
		ch.Emit("connection", []byte("new connection come"))
	})

	r.Get("/ws", wsServer.Handler)
	http.ListenAndServe(":8080", r)
}
```

## HelloWorld
```golang
package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"github.com/gobwas/ws"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	wsServer := websocket.CreateAndRun()

	r.Get("/ws", wsServer.Handler)
	wsServer.OnMessage(func(c *websocket.Conn, h ws.Header, b []byte) {
		c.Send("Hello World")
	})

	http.ListenAndServe(":8080", r)
}
```

# Benchmark
## Autobahn
All tests was runned by [Autobahn WebSocket Testsuite](https://crossbar.io/autobahn/) v0.8.0/v0.10.9.
Results:

**Code** | **Name** | **Status**
--- | --- | ---
**1** | **Framing** | **Pass**
**2** | **Pings/Pongs** | **Pass**
**3** | **Reserved Bits** | **Pass**
**4** | **Opcodes** | **Pass**
**5** | **Fragmentation** | **Pass**
**6** | **UTF-8 Handling** | **Pass**
**7** | **Close Handling** | **Pass**
**9** | **Limits/Performance** | **Pass**
**10** | **Misc** | **Pass**
**12** | **WebSocket Compression (different payloads)** | **Unimplemented**
**13** | **WebSocket Compression (different parameters)** | **Unimplemented**


# Licence
[MIT License](https://github.com/exelban/websocket/blob/master/LICENSE)
