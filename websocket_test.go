package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gobwas/ws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestServer_Run(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	time.Sleep(1 * time.Millisecond)
	require.Equal(t, 1, wsServer.Count(), "weboscket must contain only 1 connection")
}

func TestServer_Shutdown(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	err := wsServer.Shutdown()
	require.NoError(t, err)

	require.Equal(t, true, wsServer.shutdown, "websocket must be shutdown")

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	_, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err)
	require.Equal(t, "websocket: bad handshake", err.Error(), "websocket must reject connection")
}

func TestServer_Handler(t *testing.T) {
	wsServer := CreateAndRun()
	r := chi.NewRouter()

	r.Use(middleware.Compress(6, "gzip"))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, 1)
			next.ServeHTTP(ww, r)
		})
	})

	r.Get("/ws", wsServer.Handler)

	ts := httptest.NewServer(r)
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.Error(t, err, "must be rejected upgrade")
}

func TestServer_Count(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	rand.Seed(time.Now().Unix())
	number := rand.Intn(14-3) + 3

	for i := 1; i <= number; i++ {
		u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
		_, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		require.NoError(t, err)
	}

	require.Equal(t, number, wsServer.Count(), fmt.Sprintf("weboscket must contain only %d connection", number))
}

func TestServer_OnConnect(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := Message{
		Name: "TesT",
		Body: "Hello World",
	}

	wsServer.OnConnect(func(c *Conn) {
		err := c.Emit(msg.Name, msg.Body)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
		defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	for {
		var message Message
		err := c.ReadJSON(&message)
		require.NoError(t, err)

		require.Equal(t, msg, message, "response message must be the same as send")
		break
	}
}

func TestServer_OnConnect2(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := []byte("Hello from byte array")
	h := ws.Header{
		OpCode: ws.OpText,
		Fin:    true,
		Length: int64(len(msg)),
	}

	wsServer.OnConnect(func(c *Conn) {
		err := c.Write(h, msg)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
		defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	for {
		_, b, _ := c.ReadMessage()
		require.Equal(t, msg, b, "response message must be the same as send (byte array)")
		break
	}
}

func TestServer_OnDisconnect(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()
	done := make(chan bool, 1)

	msg := Message{
		Name: "TesT",
		Body: []byte("Hello World"),
	}

	wsServer.OnDisconnect(func(c *Conn) {
		_ = c.Emit(msg.Name, msg.Body)
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = c.WriteControl(8, nil, time.Now().Add(30*time.Second))
	require.NoError(t, err)

	for {
		_, b, _ := c.ReadMessage()
		require.Empty(t, b)
		break
	}

	<-done
	time.Sleep(1 * time.Millisecond)
	require.Equal(t, 0, wsServer.Count(), "server must have 0 connections")
}

func TestServer_OnMessage(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := []byte("Hello from byte array")

	done := make(chan bool, 1)
	wsServer.OnMessage(func(c *Conn, h ws.Header, b []byte) {
		require.Equal(t, msg, b, "response message must be the same as send")
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
		defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = c.WriteMessage(1, msg)
	require.NoError(t, err)

	<-done
}

func TestServer_On(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	message := Message{
		Name: "LoL",
		Body: "Hello World",
	}

	done := make(chan bool, 1)

	wsServer.On("LoL", func(c *Conn, msg *Message) {
		require.Equal(t, message, *msg, "received message must be the same as send")
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
		defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = c.WriteJSON(message)
	require.NoError(t, err)

	<-done
}

func TestServer_NewChannel(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	ch := wsServer.NewChannel("test")
	require.NotNil(t, ch, "must be channel")

	var typ *Channel
	require.IsType(t, typ, ch, "must be Channel type")
}

func TestServer_Emit(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
		defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := Message{
		Name: "test",
		Body: "Hello from emit test",
	}

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
		defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	wsServer.Emit(msg.Name, msg.Body)

	for {
		_, b, _ := c.ReadMessage()
		var res Message
		err := json.Unmarshal(b, &res)
		require.Nil(t, err, "error must be nil")
		require.Equal(t, msg, res, "response message must be the same as send (byte array)")
		break
	}
}

func TestServerListen(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err, "connection must be established without error")
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	done := make(chan bool, 1)
	message := Message{
		Name: "echo",
		Body: "Hello from echo",
	}
	wsServer.On("echo", func(c *Conn, msg *Message) {
		require.Equal(t, message, *msg, "response message must be the same as send (byte array)")
		done <- true
	})
	err = c.WriteJSON(message)
	require.NoError(t, err)
	<-done
}

func TestServerNotFound(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := []byte("Hello World")

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = c.WriteMessage(1, msg)
	require.NoError(t, err)

	for {
		_, b, _ := c.ReadMessage()
		require.Equal(t, msg, b, "response message must be the same as send (byte array)")
		break
	}
}

func TestServerProcessMessage(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	msg := []byte("")

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = c.WriteMessage(1, msg)
	require.NoError(t, err)

	for {
		_, b, _ := c.ReadMessage()
		require.Len(t, b, 0, "response length must be 0")
		break
	}
}

func wsServer() (*httptest.Server, *Server) {
	wsServer := CreateAndRun()
	r := chi.NewRouter()

	r.Get("/ws", wsServer.Handler)

	ts := httptest.NewServer(r)
	return ts, wsServer
}
