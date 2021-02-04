package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
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
	c, _, _, err := ws.Dial(context.Background(), u.String())
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

	require.Equal(t, true, wsServer.IsClosed(), "websocket must be closed")
}

func TestServer_Shutdown_byContext(t *testing.T) {
	ctx := context.Background()
	ctxWithCancel, cancel := context.WithCancel(ctx)

	wsServer := Start(ctxWithCancel)
	r := chi.NewRouter()
	r.Get("/ws", wsServer.Handler)
	ts := httptest.NewServer(r)
	defer ts.Close()

	cancel()
	time.Sleep(10 * time.Millisecond)

	require.Equal(t, true, wsServer.IsClosed(), "websocket must be closed")
}

func TestServer_Handler(t *testing.T) {
	wsServer := Start(context.Background())
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
	_, _, _, err := ws.Dial(context.Background(), u.String())
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

	connections := make([]net.Conn, 0)

	for i := 1; i <= number; i++ {
		u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
		c, _, _, err := ws.Dial(context.Background(), u.String())
		require.NoError(t, err)
		connections = append(connections, c)
	}

	require.Equal(t, number, len(connections), fmt.Sprintf("weboscket must contain only %d connection", number))
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
		Data: []byte("Hello World"),
	}
	messageBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	wsServer.OnConnect(func(c *Conn) {
		time.Sleep(300 * time.Millisecond)
		err := c.Emit(msg.Name, msg.Data)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	err = c.SetDeadline(time.Now().Add(3000 * time.Millisecond))
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	for {
		mes, op, err := wsutil.ReadServerData(c)
		require.NoError(t, err)
		require.Equal(t, true, op.IsData())
		require.Equal(t, messageBytes, mes, "response and request must be the same")

		var _message Message
		require.NoError(t, json.Unmarshal(mes, &_message))
		require.Equal(t, msg, _message, "response message must be the same as send")
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
		time.Sleep(300 * time.Millisecond)
		err := c.Write(h, msg)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	err = c.SetDeadline(time.Now().Add(3000 * time.Millisecond))
	require.NoError(t, err)

	for {
		mes, op, err := wsutil.ReadServerData(c)
		require.NoError(t, err)
		require.Equal(t, true, op.IsData())
		require.Equal(t, msg, mes, "response and request must be the same")
		break
	}

	err = c.Close()
	require.NoError(t, err)
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
		Data: []byte("Hello World"),
	}

	wsServer.OnDisconnect(func(c *Conn) {
		time.Sleep(300 * time.Millisecond)
		_ = c.Emit(msg.Name, msg.Data)
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpClose,
		Masked: true,
		Length: 0,
	})
	require.NoError(t, err)

	for {
		b := make([]byte, messagePrefix)
		err = c.SetDeadline(time.Now().Add(300 * time.Millisecond))
		require.NoError(t, err)
		_, err = c.Read(b)
		require.Error(t, err)
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
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(msg)),
	})
	require.NoError(t, err)

	n, err := c.Write(msg)
	require.NoError(t, err)
	require.Equal(t, len(msg), n)

	<-done
}

func TestServer_On(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		require.NoError(t, wsServer.Shutdown())
	}()

	type dataStruct struct {
		Test string `json:"test"`
	}
	var data = dataStruct{
		Test: "test",
	}
	_message := struct {
		Name string      `json:"name"`
		Data interface{} `json:"data"`
	}{
		Name: "LoL",
		Data: data,
	}

	messageBytes, err := json.Marshal(_message)
	require.NoError(t, err)

	done := make(chan bool, 1)

	wsServer.On("LoL", func(c *Conn, msg *Message) {
		require.Equal(t, _message.Name, msg.Name)
		var respData dataStruct
		require.NoError(t, json.Unmarshal(msg.Data, &respData))
		require.Equal(t, _message.Data, respData)
		done <- true
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(messageBytes)),
	})
	require.NoError(t, err)
	n, err := c.Write(messageBytes)
	require.NoError(t, err)
	require.Equal(t, len(messageBytes), n)

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
		Data: []byte("Hello from emit test"),
	}
	messageBytes, err := json.Marshal(msg)
	require.NoError(t, err)

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	err = c.SetDeadline(time.Now().Add(3000 * time.Millisecond))
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	wsServer.Emit(msg.Name, msg.Data)

	for {
		mes, op, err := wsutil.ReadServerData(c)
		require.NoError(t, err)
		require.Equal(t, true, op.IsData())
		require.Equal(t, messageBytes, mes, "response and request must be the same")

		var message Message
		err = json.Unmarshal(mes, &message)
		require.Equal(t, msg, message, "response message must be the same as send")
		break
	}
}

func TestServer_Channel(t *testing.T) {
	// TODO
}

func TestServerListen(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		require.NoError(t, wsServer.Shutdown())
	}()

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err, "connection must be established without error")
	defer func() {
		require.NoError(t, c.Close())
	}()

	done := make(chan bool, 1)

	message := struct {
		Name string      `json:"name"`
		Data interface{} `json:"data"`
	}{
		Name: "echo",
		Data: "Hello from echo",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err)

	wsServer.On("echo", func(c *Conn, msg *Message) {
		require.Equal(t, message.Name, msg.Name)
		var respData string
		require.NoError(t, json.Unmarshal(msg.Data, &respData))
		require.Equal(t, message.Data, respData)
		done <- true
	})

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(messageBytes)),
	})
	require.NoError(t, err)

	n, err := c.Write(messageBytes)
	require.NoError(t, err)
	require.Equal(t, len(messageBytes), n)

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
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(msg)),
	})
	require.NoError(t, err)

	n, err := c.Write(msg)
	require.NoError(t, err)
	require.Equal(t, len(msg), n)

	for {
		mes, op, err := wsutil.ReadServerData(c)
		require.NoError(t, err)
		require.Equal(t, true, op.IsData())
		require.Equal(t, msg, mes, "response message must be the same as send (byte array)")
		break
	}

	err = c.Close()
	require.NoError(t, err)
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
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(msg)),
	})
	require.NoError(t, err)

	n, err := c.Write(msg)
	require.NoError(t, err)
	require.Equal(t, len(msg), n)

	for {
		b := make([]byte, len(msg)+messagePrefix)
		err = c.SetDeadline(time.Now().Add(1000 * time.Millisecond))
		require.NoError(t, err)
		_, err = c.Read(b)
		require.NoError(t, err)
		require.Equal(t, msg, b[messagePrefix:], "response message must be the same as send (byte array)")
		break
	}
}

func wsServer() (*httptest.Server, *Server) {
	wsServer := Start(context.Background())
	r := chi.NewRouter()

	r.Get("/ws", wsServer.Handler)

	ts := httptest.NewServer(r)
	return ts, wsServer
}
