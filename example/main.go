package main

import (
	"context"
	"fmt"
	"github.com/exelban/websocket"
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type Test struct {
	Id string
}

func main () {
	var srv *http.Server

	r := chi.NewRouter()
	server := websocket.CreateAndRun()

	ch := server.NewChannel("test")

	r.Get("/ws", server.Handler)

	server.OnConnect(func(c *websocket.Conn) {
		ch.Add(c)
		ch.Emit(&websocket.Message{Name: "test", Body: fmt.Sprintf("Hello World to %d connections", ch.Count())})
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		server.Shutdown()
		srv.Shutdown(ctx)
		cancel()
		log.Print("[INFO] server stopped")
	}()

	srv = &http.Server{
		Addr: ":8080",
		Handler: r,
	}

	log.Print("[INFO] server started on port :8080")
	srv.ListenAndServe()
}
