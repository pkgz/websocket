package websocket

import (
	"sync"
)

// Channel represent group of connections (similar to group in socket.io).
type Channel struct {
	id          string
	connections map[*Conn]bool
	delConn     chan *Conn

	mu sync.RWMutex
}

func newChannel(id string) *Channel {
	c := Channel{
		id:          id,
		connections: make(map[*Conn]bool),
		delConn:     make(chan *Conn),
	}

	go func() {
		for {
			select {
			case conn := <-c.delConn:
				c.mu.Lock()
				delete(c.connections, conn)
				c.mu.Unlock()
			}
		}
	}()

	return &c
}

// Count return number of connections in channel.
func (c *Channel) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.connections)
}

// ID return channel id.
func (c *Channel) ID() string {
	return c.id
}

// Add add connection to channel.
func (c *Channel) Add(conn *Conn) {
	c.mu.Lock()
	c.connections[conn] = true
	c.mu.Unlock()
}

// Remove remove connection from channel.
func (c *Channel) Remove(conn *Conn) {
	c.mu.Lock()
	delete(c.connections, conn)
	c.mu.Unlock()
}

// Emit emits message to all connections in channel.
func (c *Channel) Emit(name string, body []byte) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for con := range c.connections {
		go func(con *Conn) {
			con.Emit(name, body)
		}(con)
	}

	return nil
}
