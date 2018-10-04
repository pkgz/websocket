package websocket

import (
	"github.com/gorilla/websocket"
	"time"
)

type Conn struct {
	id 			string
	socket 		*websocket.Conn
}

const (
	StatusLive = 1
	StatusUnknown = 2
	StatusNotResponse = 3
	StatusClosed = 4
)

func (c *Conn) Emit () error {
	return nil
}

func (c *Conn) Ping () {
	c.socket.WriteControl(websocket.PingMessage, nil, time.Time{})
}
func (c *Conn) Pong () {
	c.socket.WriteControl(websocket.PongMessage, nil, time.Time{})
}
func (c *Conn) Close (code int, message string) {
	c.socket.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, message), time.Time{})
}