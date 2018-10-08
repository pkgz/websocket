package main

import (
	"context"
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main () {
	var srv *http.Server

	r := chi.NewRouter()
	server := websocket.CreateAndRun()
	channel, _ := server.NewChannel("test")

	server.OnConnect(func(c *websocket.Conn) {
		err := channel.Add(c)
		if err != nil {
			log.Print(err)
			return
		}
	})

	r.Get("/ws", server.Handler)
	server.On("test", func(c *websocket.Conn, msg *websocket.Message) {
		c.Emit("WoW", "Test")
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		server.Shutdown()
		srv.Shutdown(ctx)
		cancel()
	}()

	srv = &http.Server{
		Addr: ":8080",
		Handler: r,
	}
	srv.ListenAndServe()
}
