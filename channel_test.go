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
		Data: "message",
	}
	messageBytes, err := json.Marshal(message)
	require.NoError(t, err)

	wsServer.OnConnect(func(c *Conn) {
		ch.Add(c)
		time.Sleep(300 * time.Millisecond)
		ch.Emit(message.Name, message.Data)
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

		var message2 Message
		err = json.Unmarshal(mes, &message2)
		require.Equal(t, message, message2, "response message must be the same as send")
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
