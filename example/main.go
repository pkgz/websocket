package main

import (
	"context"
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main () {
	var srv *http.Server

	r := chi.NewRouter()
	ws := websocket.CreateAndRun()

	r.Get("/ws", ws.Handler)
	ws.On("test", func(c *websocket.Conn, msg *websocket.Message) {
		c.Emit("WoW", "Test")
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		ws.Shutdown()
		srv.Shutdown(ctx)
		cancel()
	}()

	srv = &http.Server{
		Addr: ":8080",
		Handler: r,
	}
	srv.ListenAndServe()
}
