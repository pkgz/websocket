package websocket

import (
	"encoding/json"
	"sync"
	"time"
)

type Channel struct {
	id 						string
	connections 			map[*Conn]bool

	addConn					chan *Conn
	delConn					chan *Conn

	mu						sync.Mutex
}

func newChannel (id string) *Channel {
	c := Channel{
		id: id,
		connections: make(map[*Conn]bool),
		addConn: make(chan *Conn),
		delConn: make(chan *Conn),
	}

	go func() {
		for {
			select {
			case conn := <- c.addConn:
				c.connections[conn] = true
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
	c.addConn <- conn
	time.Sleep(100000 * time.Nanosecond)
}

// Remove remove connection from channel.
func (c *Channel) Remove (conn *Conn) {
	c.delConn <- conn
}

// Emit emits message to all connections in channel.
func (c *Channel) Emit (msg *Message) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	for con := range c.connections {
		go func(con *Conn) {
			con.Write(b)
		}(con)
	}

	return nil
}