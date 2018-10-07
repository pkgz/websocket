package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"net"
)

type Conn struct {
	id 			string
	conn 		net.Conn
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