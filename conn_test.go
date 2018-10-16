package websocket

import (
	"context"
	"github.com/gobwas/ws"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestConn_Ping(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer wsServer.Shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	m := []byte("ping")
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPing,
		Masked: true,
		Length: int64(len(m)),
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m)

	time.Sleep(1 * time.Millisecond)
	resp := make([]byte, len(m)+2)
	c.Read(resp)
	require.Equal(t, m, resp[2:], "response and request must be the same")
}

func TestConn_Pong(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer wsServer.Shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	m := []byte("pong")
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPong,
		Masked: true,
		Length: int64(len(m)),
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m)

	time.Sleep(1 * time.Millisecond)
	require.NoError(t, err)
}

func TestConn_Send(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer wsServer.Shutdown()

	msg := "ping"
	wsServer.OnConnect(func(c *Conn) {
		c.Send(msg)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	var m interface{}
	c.ReadJSON(&m)

	require.Equal(t, msg, m, "response and request must be the same")
}

func TestConn_Fragment(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer wsServer.Shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	m0 := []byte("something")
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(m0)),
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m0)

	m := []byte("nothing")
	h = ws.Header{
		Fin:    true,
		OpCode: ws.OpContinuation,
		Masked: true,
		Length: 0,
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m)

	time.Sleep(1 * time.Millisecond)

	resp := make([]byte, len(m0)+2)
	c.Read(resp)
	require.Equal(t, m0, resp[2:], "response and request must be the same")
}
