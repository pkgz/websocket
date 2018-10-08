package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/kjk/betterguid"
	"net"
	"sync"
)

type Conn struct {
	id 			string
	conn 		net.Conn
	channels	map[*channel]bool

	mu			sync.Mutex
}

func NewConn (conn net.Conn) (*Conn, error) {
	return &Conn{
		id: betterguid.New(),
		conn: conn,
		channels: make(map[*channel]bool),
	}, nil
}

// Id return connection internal id
func (c *Conn) Id () string{
	return c.id
}

// Join connect to specific channel
func (c *Conn) Join (ch *channel) error {
	if c.channels[ch] {
		return errors.New(fmt.Sprintf("websocket: connection already in channel %s", ch.Id()))
	}

	c.mu.Lock()
	c.channels[ch] = true
	c.mu.Unlock()

	return nil
}

// Join will remove connection from specific channel
func (c *Conn) Leave (ch *channel) error {
	if !c.channels[ch] {
		return errors.New(fmt.Sprintf("websocket: connection not in channel %s", ch.Id()))
	}

	c.mu.Lock()
	delete(c.channels, ch)
	c.mu.Unlock()

	return nil
}

// Emit allow to send message to connection
func (c *Conn) Emit (name string, body interface{}) error {
	msg := Message{
		Name: name,
		Body: fmt.Sprintf("%v", body),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = wsutil.WriteServerMessage(c.conn, ws.OpText, b)
	if err != nil {
		return err
	}

	return nil
}

// Default handler for pong request
func (c *Conn) Ping () {
	wsutil.WriteServerMessage(c.conn, ws.OpPing, []byte("ping"))
}

// Default handler for ping request
func (c *Conn) Pong () {
	wsutil.WriteServerMessage(c.conn, ws.OpPong, []byte("pong"))
}

// Close connection
func (c *Conn) Close () {
	wsutil.WriteServerMessage(c.conn, ws.OpClose, nil)
}

// Close connection with status
func (c *Conn) CloseWithStatus (code ws.StatusCode, reason string) {
	closeProtocol := ws.MustCompileFrame(
		ws.NewCloseFrame(ws.NewCloseFrameBody(
			code, reason,
		)),
	)
	c.conn.Write(closeProtocol)
}