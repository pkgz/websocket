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