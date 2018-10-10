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