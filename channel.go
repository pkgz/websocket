package websocket

import (
	"sync"
)

type Channel struct {
	id 						string
	connections 			map[*Conn]bool
	delConn					chan *Conn

	mu						sync.Mutex
}

func newChannel (id string) *Channel {
	c := Channel{
		id: id,
		connections: make(map[*Conn]bool),
		delConn: make(chan *Conn),
	}

	go func() {
		for {
			select {
			case conn := <- c.delConn:
				delete(c.connections, conn)
			}
		}
	}()

	return &c
}

// Count return number of connections in channel.
func (c *Channel) Count () int {
	return len(c.connections)
}

// Id return channel id.
func (c *Channel) Id () string {
	return c.id
}

// Add add connection to channel.
func (c *Channel) Add (conn *Conn) {
	c.mu.Lock()
	c.connections[conn] = true
	c.mu.Unlock()
}

// Remove remove connection from channel.
func (c *Channel) Remove (conn *Conn) {
	c.mu.Lock()
	delete(c.connections, conn)
	c.mu.Unlock()
}

// Emit emits message to all connections in channel.
func (c *Channel) Emit (name string, body []byte) error {
	for con := range c.connections {
		go func(con *Conn) {
			con.Emit(name, body)
		}(con)
	}

	return nil
}