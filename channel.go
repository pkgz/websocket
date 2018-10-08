package websocket

import (
	"fmt"
	"github.com/pkg/errors"
	"sync"
)

type Channel struct {
	id 				string
	connections 	map[*Conn]bool

	closed			bool
	mu				sync.Mutex
}

// Emit emits message to all connections
func CreateChannel (name string) (*Channel, error) {
	return &Channel{
		id: name,
		connections: make(map[*Conn]bool),
	}, nil
}

// Add add connection to channel
func (c *Channel) Add (conn *Conn) error {
	if c.closed {
		return errors.New(fmt.Sprintf("websocket: trying to add connection to closed group %s", c.Id()))
	}
	conn.Join(c)

	c.mu.Lock()
	c.connections[conn] = true
	c.mu.Unlock()

	return nil
}

// Remove remove connection from channel
func (c *Channel) Remove (conn *Conn) error {
	if c.closed {
		return errors.New(fmt.Sprintf("websocket: trying to leave group %s, which is already deleted", c.Id()))
	}
	conn.Leave(c)

	if !c.connections[conn] {
		return errors.New("websocket: connection not find in this channel")
	}

	c.mu.Lock()
	delete(c.connections, conn)
	c.mu.Unlock()

	return nil
}

// Id return channel name which is id
func (c *Channel) Id () string {
	return c.id
}

// Emit emits message to all connections
func (c *Channel) Emit (name string, message interface{}) error {
	if c.closed {
		return errors.New(fmt.Sprintf("websocket: trying to emit message in deleted group %s", c.Id()))
	}

	for conn := range c.connections {
		conn.Emit(name, message)
	}

	return nil
}

// Delete delete channel and remove all connections from it
func (c *Channel) Delete () error {
	if c.closed {
		return errors.New(fmt.Sprintf("websocket: trying to delete group %s, which is already deleted", c.Id()))
	}

	for conn := range c.connections {
		conn.Leave(c)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true

	return nil
}