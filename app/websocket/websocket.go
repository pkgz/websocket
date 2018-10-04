package sockets

import (
	"net/http"
	"sync"
)

type Sockets struct {
	clients 		map[*Conn]bool
	broadcast 		chan interface{}
	handlers 		map[string]func(body []byte)

	lock 			sync.Mutex
}


type InterfaceS interface {
	New() *Sockets
	NewBroadcaster() error
	Handler()
	Close()
	On()
	Emit()
}


func New() *Sockets {
	return &Sockets{
		clients: make(map[*Conn]bool),
		broadcast: make(chan interface{}),
		handlers: make(map[string]func(body []byte)),
	}
}

func (s *Sockets) Handler (w http.ResponseWriter, r *http.Request) {}

func (s *Sockets) On (name string, ) {}