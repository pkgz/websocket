package websocket

import (
	"github.com/gobwas/ws"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestConn_Ping(t *testing.T) {
	server, wsServer, ctx := createWS()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	m := []byte("ping")
	mask := 6
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPing,
		Masked: true,
		Length: int64(len(m)),
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m)

	resp := make([]byte, mask + len(m))
	c.Read(resp)
	require.Equal(t, m, resp[mask:mask+ len(m)], "response and request must be the same")

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestConn_Pong(t *testing.T) {
	server, wsServer, ctx := createWS()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, _, err := ws.Dial(ctx, u.String())
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	m := []byte("pong")
	mask := 6
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPong,
		Masked: true,
		Length: int64(len(m)),
	}
	ws.WriteHeader(c, h)
	_, err = c.Write(m)

	resp := make([]byte, mask + len(m))
	c.Read(resp)
	require.Equal(t, m, resp[mask:mask+ len(m)], "response and request must be the same")

	wsServer.Shutdown()
	server.Shutdown(ctx)
}