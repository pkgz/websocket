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

func TestChannel_Add(t *testing.T) {
	ts, wsServer, shutdown := server(t)
	defer shutdown()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(ctx context.Context, c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()
}

func TestChannel_Emit(t *testing.T) {
	ts, wsServer, shutdown := server(t)
	defer shutdown()

	ch := wsServer.NewChannel("test-channel-emit")

	_message := Message{
		Name: "test-channel-emit",
		Data: []byte("message"),
	}
	messageBytes, err := json.Marshal(_message)
	require.NoError(t, err)

	wsServer.OnConnect(func(ctx context.Context, c *Conn) {
		ch.Add(c)
		time.Sleep(300 * time.Millisecond)
		ch.Emit(_message.Name, _message.Data)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	err = c.SetDeadline(time.Now().Add(3000 * time.Millisecond))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()

	for {
		mes, op, err := wsutil.ReadServerData(c)
		require.NoError(t, err)
		require.Equal(t, true, op.IsData())
		require.Equal(t, messageBytes, mes, "response and request must be the same")

		var msg Message
		require.NoError(t, json.Unmarshal(mes, &msg))
		require.Equal(t, _message, msg, "response message must be the same as send")
		break
	}
}

func TestChannel_Remove(t *testing.T) {
	ts, wsServer, shutdown := server(t)
	defer shutdown()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(ctx context.Context, c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
		ch.Remove(c)
		require.Equal(t, 0, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		require.NoError(t, c.Close())
	}()
}

func TestChannel_Id(t *testing.T) {
	_, wsServer, shutdown := server(t)
	defer shutdown()

	ch := wsServer.NewChannel("test-channel-id")
	require.Equal(t, "test-channel-id", ch.ID(), "channel must have same id")
}
