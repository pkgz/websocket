// Package websocket implements the WebSocket protocol defined in RFC 6455.
package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"reflect"
	"sync"
	"unicode/utf8"
)

// Server allow to keep connection list, broadcast channel and callbacks list
type Server struct {
	connections 	map[*Conn]bool
	broadcast 		chan interface{}
	channels		map[string]*channel

	callbacks 		map[string]HandlerFunc

	onConnect		func(c *Conn)

	done 			chan bool
	mu 				sync.Mutex
}

// Message is a struct for data which sending between application and clients
// Name using for matching callback function in On function
// Body will be transformed to byte array and returned to callback
type Message struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

// HamdleFunc is a type for handle function
// all function which has callback have this struct
// as first element returns pointer to connection
// its give oportunity to close connection or emit message to exactly this connection
type HandlerFunc func (c *Conn, msg *Message)

// Create a new websocket server handler with the provided options.
func Create() *Server {
	return &Server{
		connections: make(map[*Conn]bool),
		broadcast: make(chan interface{}),
		channels: make(map[string]*channel),
		callbacks: make(map[string]HandlerFunc),
		done: make(chan bool, 1),
	}
}

// CreateAndRun instantly create and run websocket server
func CreateAndRun() *Server {
	srv := Create()
	srv.Run()

	return srv
}

// Run starting goroutines with broadcaster loop
func (s *Server) Run() {
	go s.Broadcast()
}

// Handler get upgrade connection to RFC 6455 and starting listener for it
func (s *Server) Handler (w http.ResponseWriter, r *http.Request) {
	con, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("connection cannot be upgraded to websocket"))
		return
	}
	defer con.Close()

	connection, err := NewConn(con)

	s.mu.Lock()
	s.connections[connection] = true
	s.mu.Unlock()

	if !reflect.ValueOf(s.onConnect).IsNil() {
		s.onConnect(connection)
	}

	s.listener(connection)
}

// Set function which will be called when new connections come
func (s *Server) OnConnect (f func(c *Conn)) {
	s.mu.Lock()
	s.onConnect = f
	s.mu.Unlock()
}

// Adding callback for specific message name
func (s *Server) On (name string, f HandlerFunc) {
	s.mu.Lock()
	s.callbacks[name] = f
	s.mu.Unlock()
}

// Sending message to all connections.
// For sending message to specific connection use Conn.Emit()
func (s *Server) Emit (name string, body interface{}) {
	s.broadcast <- &Message{
		Name: name,
		Body: fmt.Sprintf("%v", body),
	}
}

// Broadcast forever loop which waiting for new message on broadcast channel
// if new message pushed to channel it will be send to connection
// also waiting for done event
func (s *Server) Broadcast() error {
	for {
		select {
		case <- s.done:
			return nil
		default:
			msg := <- s.broadcast
			for c := range s.connections {
				b, err := json.Marshal(msg)
				if err != nil {
					s.closeConnection(c, ws.StatusInternalServerError, err.Error())
				}
				err = wsutil.WriteServerMessage(c.conn, ws.OpText, b)
				if err != nil {
					s.closeConnection(c, ws.StatusInternalServerError, err.Error())
				}
			}
		}
	}
	return nil
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping goroutines with broadcaster loop
func (s *Server) Shutdown() error {
	l := len(s.connections)
	var wg sync.WaitGroup
	wg.Add(l)

	for c := range s.connections {
		go func(c *Conn) {
			s.closeConnection(c, ws.StatusNormalClosure, "")
			wg.Done()
		}(c)
	}

	wg.Wait()
	s.done <- true

	return nil
}

// NewChannel will create new channel with specific name.
// Return error if something goes wrong.
// Channel must have unique name!
func (s *Server) NewChannel (name string) (*channel, error) {
	if s.channels[name] != nil {
		return nil, errors.New(fmt.Sprintf("websocket: channel with name %s, already exist", name))
	}

	c, err := CreateChannel(name)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.channels[name] = c
	s.mu.Unlock()

	return c, nil
}

// Count return number of active connections
func (s *Server) Count () int {
	return len(s.connections)
}

func (s *Server) listener (c *Conn) {
	for {
		b, op, err := wsutil.ReadClientData(c.conn)
		log.Print(op)

		if err != nil {
			s.mu.Lock()
			delete(s.connections, c)
			s.mu.Unlock()
			break
		}

		switch op {
		case ws.OpContinuation:
			continue
		case ws.OpText:
			if !utf8.Valid(b) {
				s.closeConnection(c, ws.StatusUnsupportedData, "Unsupported Data")
				return
			}
			s.processMessage(c, b)
		case ws.OpBinary:
			s.closeConnection(c, ws.StatusUnsupportedData, "Unsupported Data")
		case ws.OpClose:
			s.closeConnection(c, ws.StatusNormalClosure, "")
		case ws.OpPing:
			c.Pong()
		case ws.OpPong:
			c.Ping()
		}
	}
}

func (s *Server) processMessage (c *Conn, b []byte) {
	var msg Message
	err := json.Unmarshal(b, &msg)
	if err != nil {
		log.Printf("websocket: wrong message structure: %+v", err)
		return
	}

	if s.callbacks[msg.Name] != nil {
		s.callbacks[msg.Name](c, &msg)
		return
	}

	s.notFound(c, &msg)
}

func (s *Server) closeConnection (c *Conn, code ws.StatusCode, reason string) error {
	c.CloseWithStatus(code, reason)

	if !s.connections[c] {
		return errors.New("websocket: unable to find connection")
	}

	s.mu.Lock()
	delete(s.connections, c)
	s.mu.Unlock()

	return nil
}

func (s *Server) notFound(c *Conn, msg *Message) {
	c.Emit("not found", msg.Body)
}