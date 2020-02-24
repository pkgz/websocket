# websocket

[![GoDoc](http://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/pkgz/websocket)
[![Go Report Card](https://goreportcard.com/badge/github.com/pkgz/websocket)](https://goreportcard.com/report/github.com/pkgz/websocket)
[![Tests](https://img.shields.io/github/workflow/status/pkgz/websocket/Code%20coverage)](https://github.com/pkgz/websocket/actions)
[![codecov](https://img.shields.io/codecov/c/gh/pkgz/websocket)](https://codecov.io/gh/pkgz/websocket)

Simple websocket library for golang

## Installation
```bash
go get github.com/pkgz/websocket
```

## Example
### Echo
```golang
package main

import (
	"context"
	"github.com/pkgz/websocket"
	"net/http"
)

func main() {
	wsServer := websocket.Start(context.Background())

	r := http.NewServeMux()
	r.HandleFunc("/ws", wsServer.Handler)

	wsServer.On("echo", func(c *websocket.Conn, msg *websocket.Message) {
		_ = c.Emit("echo", msg.Body)
	})

	_ = http.ListenAndServe(":8080", r)
}
```

### Channel
```golang
package main

import (
	"context"
	"github.com/pkgz/websocket"
	"net/http"
)

func main() {
	wsServer := websocket.Start(context.Background())

	r := http.NewServeMux()
	r.HandleFunc("/ws", wsServer.Handler)

	ch := wsServer.NewChannel("test")
	wsServer.OnConnect(func(c *websocket.Conn) {
		ch.Add(c)
		ch.Emit("connection", "new connection come")
	})

	_ = http.ListenAndServe(":8080", r)
}
```

### Hello World
```golang
package main

import (
	"context"
	"github.com/pkgz/websocket"
	"github.com/gobwas/ws"
	"net/http"
)

func main () {
	r := http.NewServeMux()
	wsServer := websocket.Start(context.Background())

	r.HandleFunc("/ws", wsServer.Handler)
	wsServer.OnMessage(func(c *websocket.Conn, h ws.Header, b []byte) {
		c.Send("Hello World")
	})

	http.ListenAndServe(":8080", r)
}
```

## Benchmark
### Autobahn
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

## Licence
[MIT License](https://github.com/pkgz/websocket/blob/master/LICENSE)
