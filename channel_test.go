package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestChannel_Add(t *testing.T) {
	server, wsServer, ctx := createWS()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestChannel_Emit(t *testing.T) {
	server, wsServer, ctx := createWS()

	ch := wsServer.NewChannel("test-channel-emit")

	message := Message{
		Name: "test-channel-emit",
		Body: []byte("message"),
	}

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		ch.Emit(message.Name, message.Body)
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	for {
		var msg Message
		c.ReadJSON(&msg)
		require.Equal(t, message, msg, "received message must be same as send")
		break
	}

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestChannel_Remove(t *testing.T) {
	server, wsServer, ctx := createWS()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
		ch.Remove(c)
		require.Equal(t, 0, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	wsServer.Shutdown()
	server.Shutdown(ctx)
}

func TestChannel_Id(t *testing.T) {
	server, wsServer, ctx := createWS()

	ch := wsServer.NewChannel("test-channel-id")
	require.Equal(t, "test-channel-id", ch.ID(), "channel must have same id")

	wsServer.Shutdown()
	server.Shutdown(ctx)
}
