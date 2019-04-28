package websocket

import (
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
	"testing"
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
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
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

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		err := ch.Emit(message.Name, message.Body)
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
		var msg Message
		err := c.ReadJSON(&msg)
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
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
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
