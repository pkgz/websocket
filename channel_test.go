package websocket

import (
	"context"
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestChannel_Add(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
}

func TestChannel_Emit(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	ch := wsServer.NewChannel("test-channel-emit")

	message := Message{
		Name: "test-channel-emit",
		Body: "message",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err)

	wsServer.OnConnect(func(c *Conn) {
		time.Sleep(100 * time.Millisecond)
		ch.Add(c)
		err := ch.Emit(message.Name, message.Body)
		require.NoError(t, err)
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()

	for {
		var msg Message
		b := make([]byte, len(messageBytes)+messagePrefix)
		n, err := c.Read(b)
		require.NoError(t, err)
		err = json.Unmarshal(b[messagePrefix:n], &msg)
		require.NoError(t, err)
		require.Equal(t, message, msg, "received message must be same as send")
		break
	}
}

func TestChannel_Remove(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	ch := wsServer.NewChannel("test-channel-add")

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		require.Equal(t, 1, ch.Count(), "channel must contain only 1 connection")
		ch.Remove(c)
		require.Equal(t, 0, ch.Count(), "channel must contain only 1 connection")
	})

	u := url.URL{Scheme: "ws", Host: strings.Replace(ts.URL, "http://", "", 1), Path: "/ws"}
	c, _, _, err := ws.Dial(context.Background(), u.String())
	require.NoError(t, err)
	defer func() {
		err := c.Close()
		require.NoError(t, err)
	}()
}

func TestChannel_Id(t *testing.T) {
	ts, wsServer := wsServer()
	defer ts.Close()
	defer func() {
		err := wsServer.Shutdown()
		require.NoError(t, err)
	}()

	ch := wsServer.NewChannel("test-channel-id")
	require.Equal(t, "test-channel-id", ch.ID(), "channel must have same id")
}
