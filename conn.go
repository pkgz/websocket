package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"net"
)

// Conn websocket connection
type Conn struct {
	conn net.Conn
}

// Emit emit message to connection.
func (c *Conn) Emit(name string, body []byte) error {
	msg := Message{
		Name: name,
		Body: body,
	}
	b, _ := json.Marshal(msg)

	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: false,
		Length: int64(len(b)),
	}

	return c.Write(h, b)
}

// Write write byte array to connection.
func (c *Conn) Write(h ws.Header, b []byte) error {
	err := ws.WriteHeader(c.conn, h)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(b)
	return err
}

// Ping handler for pong request.
func (c *Conn) Ping(b []byte) error {
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPing,
		Masked: true,
		Length: int64(len(b)),
	}
	err := c.Write(h, b)
	return err
}

// Pong handler for ping request.
func (c *Conn) Pong(b []byte) error {
	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpPong,
		Masked: true,
		Length: int64(len(b)),
	}
	err := c.Write(h, b)
	return err
}

// Close closing websocket connection.
func (c *Conn) Close() error {
	err := c.conn.Close()
	return err
}
