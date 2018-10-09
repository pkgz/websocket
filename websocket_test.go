package websocket

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"
)

var addr = flag.String("addr", "localhost:8080", "http service address")


func TestServer_Run(t *testing.T) {
	server, wsServer, ctx := createWS()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	require.Equal(t, 1, wsServer.Count(), "weboscket must contain only 1 connection")

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_Shutdown(t *testing.T) {
	server, wsServer, ctx := createWS()
	wsServer.Shutdown()

	require.Equal(t, true, wsServer.shutdown, "websocket must be shutdown")

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Equal(t, "websocket: bad handshake", err.Error(), "websocket must reject connection")

	server.Shutdown(ctx)
}

func TestServer_Count(t *testing.T) {
	server, wsServer, ctx := createWS()

	rand.Seed(time.Now().Unix())
	number := rand.Intn(14 - 3) + 3

	for i := 1; i <= number; i++ {
		u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
		_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal("dial:", err)
		}
	}

	require.Equal(t, number, wsServer.Count(), fmt.Sprintf("weboscket must contain only %d connection", number))

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_OnConnect(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := Message{
		Name: "TesT",
		Body: "Hello World",
	}

	wsServer.OnConnect(func(c *Conn) {
		c.Emit(&msg)
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	for {
		var message Message
		c.ReadJSON(&message)
		require.Equal(t, msg, message, "response message must be the same as send")
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_OnConnect2(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := []byte("Hello from byte array")

	wsServer.OnConnect(func(c *Conn) {
		c.Write(msg)
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	for {
		_, b, _ := c.ReadMessage()
		require.Equal(t, msg, b, "response message must be the same as send (byte array)")
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_OnDisconnect(t *testing.T) {
	server, wsServer, ctx := createWS()
	done := make(chan bool, 1)

	msg := Message{
		Name: "TesT",
		Body: "Hello World",
	}

	wsServer.OnDisconnect(func(c *Conn) {
		done <- true
		err := c.Emit(&msg)
		require.Error(t, err, "server must return error with closed connection")
		require.Equal(t, 0, wsServer.Count(), "server must have 0 connections")
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	for {
		c.WriteControl(8, nil, time.Now().Add(30 * time.Second))
		<- done
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestProcessMessage(t *testing.T) {
	server, wsServer, ctx := createWS()

	err := wsServer.processMessage(nil, nil)
	require.Error(t, err, "unexpected end of JSON input")

	err = wsServer.processMessage(nil, []byte(""))
	require.Error(t, err, "unexpected end of JSON input")

	m := Message{Name: "1", Body: "2"}
	b, err := json.Marshal(m)
	err = wsServer.processMessage(nil, b)
	require.Nil(t, err, "must normally process message")


	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func createWS () (*http.Server, *Server, context.Context) {
	var srv *http.Server

	r := chi.NewRouter()
	server := CreateAndRun()

	r.Get("/ws", server.Handler)

	ctx, _ := context.WithCancel(context.Background())
	srv = &http.Server{
		Addr: ":8080",
		Handler: r,
	}

	go func() {
		srv.ListenAndServe()
	}()

	return srv, server, ctx
}