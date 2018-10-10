package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/kjk/betterguid"
	"net"
)

type Conn struct {
	id 		string
	conn 	net.Conn
}

func NewConn (conn net.Conn) *Conn {
	return &Conn{
		id: betterguid.New(),
		conn: conn,
	}
}

// Emit emit message to connection.
func (c *Conn) Emit (name string, body []byte) error {
	msg := Message{
		Name: name,
		Body: body,
	}
	b, _ := json.Marshal(msg)

	return c.Write(b)
}

// Write write byte array to connection.
func (c *Conn) Write (b []byte) error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpText, b)
	return err
}

// Ping handler for pong request.
func (c *Conn) Ping () error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpPing, nil)
	return err
}

// Pong handler for ping request.
func (c *Conn) Pong () error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpPong, nil)
	return err
}

// Close closing websocket connection.
func (c *Conn) Close () error {
	err := c.conn.Close()
	return err
}