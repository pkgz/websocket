package websocket

import (
	"sync"
)

// Channel represent group of connections (similar to group in socket.io).
type Channel struct {
	id          string
	connections map[*Conn]bool
	delConn     chan *Conn

	mu sync.Mutex
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
				_ = conn.Close()
				delete(c.connections, conn)
				c.mu.Unlock()
			}
		}
	}()

	return &c
}

// Count return number of live connections in channel.
func (c *Channel) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for con := range c.connections {
		if con.conn != nil {
			count++
		}
	}

	return count
}

// ID return channel id.
func (c *Channel) ID() string {
	return c.id
}

// Add connection to channel.
func (c *Channel) Add(conn *Conn) {
	c.mu.Lock()
	c.connections[conn] = true
	c.mu.Unlock()
}

// Remove connection from channel.
func (c *Channel) Remove(conn *Conn) {
	c.mu.Lock()
	delete(c.connections, conn)
	c.mu.Unlock()
}

// Emit message to all connections in channel.
func (c *Channel) Emit(name string, data interface{}) {
	c.mu.Lock()

	for con := range c.connections {
		if err := con.Emit(name, data); err != nil {
			_ = con.Close()

			c.mu.Unlock()
			c.Remove(con)
			c.mu.Lock()
		}
	}

	c.mu.Unlock()
}

// Purge remove all connections from channel.
func (c *Channel) Purge() {
	c.mu.Lock()
	c.connections = make(map[*Conn]bool)
	c.delConn = make(chan *Conn)
	c.mu.Unlock()
}
