package main

import (
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
)

func main() {
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
