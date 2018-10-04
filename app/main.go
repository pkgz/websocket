package main

import (
	"github.com/exelban/websocket/app/websocket"
	"github.com/go-chi/chi"
	"log"
	"net/http"
)

func main () {
	r := chi.NewRouter()
	ws := websocket.New()
	go ws.Broadcaster()

	r.Get("/ws", ws.Handler)

	ws.On("test", func(c *websocket.Conn, mes *websocket.Message) {
		log.Print(mes)
	})

	log.Print("[INFO] server started")
	http.ListenAndServe(":8080", r)
}