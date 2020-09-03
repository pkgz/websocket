package main

import (
	"context"
	"github.com/pkgz/websocket"
	"net/http"
)

func main() {
	wsServer := websocket.Start(context.Background())

	r := http.NewServeMux()
	r.HandleFunc("/", wsServer.Handler)

	wsServer.On("echo", func(c *websocket.Conn, msg *websocket.Message) {
		_ = c.Emit("echo", msg.Data)
	})

	_ = http.ListenAndServe(":9001", r)
}
