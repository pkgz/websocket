package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"net"
	"sync"
	"time"
)

// Conn websocket connection
type Conn struct {
	conn net.Conn
	mu   sync.Mutex
}

// Emit emit message to connection.
func (c *Conn) Emit(name string, body interface{}) error {
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
	c.mu.Lock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(3000 * time.Millisecond))
	err := ws.WriteHeader(c.conn, h)
	if err != nil {
		return err
	}
	_, err = c.conn.Write(b)
	c.mu.Unlock()
	return err
}

// Send send data to connection.
func (c *Conn) Send(data interface{}) error {
	var b []byte
	switch data.(type) {
	case []byte:
		b = data.([]byte)
	default:
		b, _ = json.Marshal(data)
	}

	h := ws.Header{
		Fin:    true,
		OpCode: ws.OpText,
		Masked: false,
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
