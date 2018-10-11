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
		c.Write(h, b)
	})

	http.ListenAndServe(":8080", r)
}