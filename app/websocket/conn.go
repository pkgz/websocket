package sockets

import (
	"github.com/gorilla/websocket"
)

type Conn struct {
	id 		string
	status 	int
	s 		*websocket.Conn
}

const (
	StatusLive = 1
	StatusUnknown = 2
	StatusNotResponse = 3
	StatusClosed = 4
)

type Interface interface {
	Emit() 		error
	Ping() 		error
	Close() 	error
}