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

// Conn return net.Conn of connection
func (c *Conn) Conn () net.Conn {
	return c.conn
}

// Emit emit message to connection
func (c *Conn) Emit (msg *Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	err = wsutil.WriteServerMessage(c.conn, ws.OpText, b)
	return err
}

// Write write byte array to connection
func (c *Conn) Write (b []byte) error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpText, b)
	return err
}

// Ping handler for pong request
func (c *Conn) Ping () error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpPing, []byte("ping"))
	return err
}

// Pong handler for ping request
func (c *Conn) Pong () error {
	err := wsutil.WriteServerMessage(c.conn, ws.OpPong, []byte("pong"))
	return err
}

// Close closing websocket connection
func (c *Conn) Close () error {
	err := c.conn.Close()
	return err
}