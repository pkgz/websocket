package websocket

import (
	"context"
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
	"testing"
	"time"
)

const messagePrefix = 2

func TestConn_Ping(t *testing.T) {
	ts, _, shutdown := server(t)
	defer shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	m := []byte("ping")
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPing,
		Masked: true,
		Length: int64(len(m)),
	}
	err = ws.WriteHeader(c, h)
	require.NoError(t, err)
	_, err = c.Write(m)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	resp := make([]byte, len(m)+2)
	_, err = c.Read(resp)
	require.NoError(t, err)
	require.Equal(t, m, resp[2:], "response and request must be the same")
}

func TestConn_Pong(t *testing.T) {
	ts, _, shutdown := server(t)
	defer shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	m := []byte("pong")
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPong,
		Masked: true,
		Length: int64(len(m)),
	}
	err = ws.WriteHeader(c, h)
	require.NoError(t, err)
	_, err = c.Write(m)
	require.NoError(t, err)

	time.Sleep(1 * time.Millisecond)
	require.NoError(t, err)
}

func TestConn_Send_bytes(t *testing.T) {
	ts, wsServer, shutdown := server(t)
	defer shutdown()

	msg := []byte{0, 1, 2, 3, 4, 5, 6, 7}
	wsServer.OnConnect(func(ctx context.Context, c *Conn) {
		time.Sleep(300 * time.Millisecond)
		err := c.Send(msg)
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

	mes, op, err := wsutil.ReadServerData(c)
	require.NoError(t, err)
	require.Equal(t, true, op.IsData())
	require.Equal(t, msg, mes, "response and request must be the same")
}

func TestConn_Send_struct(t *testing.T) {
	ts, wsServer, shutdown := server(t)
	defer shutdown()

	msg := struct {
		Value string `json:"value"`
	}{
		Value: "test",
	}
	wsServer.OnConnect(func(ctx context.Context, c *Conn) {
		time.Sleep(300 * time.Millisecond)
		err := c.Send(msg)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	err = c.SetDeadline(time.Now().Add(3000 * time.Millisecond))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()

	mes, op, err := wsutil.ReadServerData(c)
	require.NoError(t, err)
	require.Equal(t, true, op.IsData())

	b, err := json.Marshal(msg)
	require.NoError(t, err)

	require.Equal(t, b, mes, "response and request must be the same")
}

func TestConn_Fragment(t *testing.T) {
	ts, _, shutdown := server(t)
	defer shutdown()

	ctx := context.Background()
	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()

	m1 := []byte("hello")
	err = ws.WriteHeader(c, ws.Header{
		Fin:    false,
		OpCode: ws.OpText,
		Masked: true,
		Length: int64(len(m1)),
	})
	require.NoError(t, err)
	_, err = c.Write(m1)
	require.NoError(t, err)

	m2 := []byte(" world")
	err = ws.WriteHeader(c, ws.Header{
		Fin:    true,
		OpCode: ws.OpContinuation,
		Masked: true,
		Length: int64(len(m2)),
	})
	require.NoError(t, err)
	_, err = c.Write(m2)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	resp := make([]byte, 128)
	n, err := c.Read(resp)
	resp = resp[:n]
	final := []byte{}
	for _, b := range resp {
		if b != 0x1 && b != 0x5 && b != 0x6 && b != 0x80 {
			final = append(final, b)
		}
	}
	require.NoError(t, err)
	require.Equal(t, append(m1, m2...), final, "response and request must be the same")
}
