package websocket

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/gobwas/ws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"
)

var addr = flag.String("addr", "127.0.0.1:8080", "http service address")

func TestServer_Run(t *testing.T) {
	server, wsServer, ctx := createWS()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	time.Sleep(1 * time.Millisecond) // Give some time to add connection

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
	number := rand.Intn(14-3) + 3

	for i := 1; i <= number; i++ {
		u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
		_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal("dial:", err)
		}
	}

	time.Sleep(1 * time.Millisecond) // Give some time to add connection
	require.Equal(t, number, wsServer.Count(), fmt.Sprintf("weboscket must contain only %d connection", number))

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_OnConnect(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := Message{
		Name: "TesT",
		Body: []byte("Hello World"),
	}

	wsServer.OnConnect(func(c *Conn) {
		c.Emit(msg.Name, msg.Body)
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
	h := ws.Header{
		OpCode: ws.OpText,
		Fin:    true,
		Length: int64(len(msg)),
	}

	wsServer.OnConnect(func(c *Conn) {
		c.Write(h, msg)
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
		Body: []byte("Hello World"),
	}

	wsServer.OnDisconnect(func(c *Conn) {
		done <- true
		err := c.Emit(msg.Name, msg.Body)
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
		c.WriteControl(8, nil, time.Now().Add(30*time.Second))
		<-done
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_OnMessage(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := []byte("Hello from byte array")

	done := make(chan bool, 1)
	wsServer.OnMessage(func(c *Conn, h ws.Header, b []byte) {
		require.Equal(t, msg, b, "response message must be the same as send")
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	c.WriteMessage(1, msg)

	<-done

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_On(t *testing.T) {
	server, wsServer, ctx := createWS()

	message := Message{
		Name: "LoL",
		Body: []byte("Hello World"),
	}

	done := make(chan bool, 1)

	wsServer.On("LoL", func(c *Conn, msg *Message) {
		require.Equal(t, message, *msg, "received message must be the same as send")
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	c.WriteJSON(message)

	<-done

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_NewChannel(t *testing.T) {
	server, wsServer, ctx := createWS()

	ch := wsServer.NewChannel("test")
	require.NotNil(t, ch, "must be channel")

	var typ *Channel
	require.IsType(t, typ, ch, "must be Channel type")

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServer_Emit(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := Message{
		Name: "test",
		Body: []byte("Hello from emit test"),
	}

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	wsServer.Emit(msg.Name, msg.Body)

	for {
		_, b, _ := c.ReadMessage()
		var res Message
		err := json.Unmarshal(b, &res)
		require.Nil(t, err, "error must be nil")
		require.Equal(t, msg, res, "response message must be the same as send (byte array)")
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServerListen(t *testing.T) {
	server, wsServer, ctx := createWS()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err, "connection must be established without error")
	defer c.Close()

	done := make(chan bool, 1)
	message := Message{
		Name: "echo",
		Body: []byte("Hello from echo"),
	}
	wsServer.On("echo", func(c *Conn, msg *Message) {
		require.Equal(t, message, *msg, "response message must be the same as send (byte array)")
		done <- true
	})
	c.WriteJSON(message)
	<-done

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestServerNotFound(t *testing.T) {
	server, wsServer, ctx := createWS()

	msg := []byte("Hello World")

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	c.WriteMessage(1, msg)

	for {
		_, b, _ := c.ReadMessage()
		require.Equal(t, msg, b, "response message must be the same as send (byte array)")
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func createWS() (*http.Server, *Server, context.Context) {
	var srv *http.Server

	r := chi.NewRouter()
	server := CreateAndRun()

	r.Get("/ws", server.Handler)

	ctx, cancel := context.WithCancel(context.Background())
	srv = &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		srv.ListenAndServe()
		cancel()
	}()

	time.Sleep(100 * time.Millisecond) // give time to server wake up
	return srv, server, ctx
}
