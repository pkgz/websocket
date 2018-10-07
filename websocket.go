// Package websocket implements the WebSocket protocol defined in RFC 6455.
package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"github.com/kjk/betterguid"
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

	callbacks 		map[string]HandlerFunc

	onConnect		func(c *Conn)

	done 			chan bool
	lock 			sync.Mutex
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
func Create() Server {
	return Server{
		connections: make(map[*Conn]bool),
		broadcast: make(chan interface{}),
		callbacks: make(map[string]HandlerFunc),
		done: make(chan bool, 1),
	}
}

// CreateAndRun instantly create and run websocket server
func CreateAndRun() Server {
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

	connection := &Conn{
		id: betterguid.New(),
		conn: con,
	}

	s.lock.Lock()
	s.connections[connection] = true
	s.lock.Unlock()

	if !reflect.ValueOf(s.onConnect).IsNil() {
		s.onConnect(connection)
	}
	s.listener(connection)
}

// Set function which will be called when new connections come
func (s *Server) OnConnect (f func(c *Conn)) {
	s.lock.Lock()
	s.onConnect = f
	s.lock.Unlock()
}

// Adding callback for specific message name
func (s *Server) On (name string, f HandlerFunc) {
	s.lock.Lock()
	s.callbacks[name] = f
	s.lock.Unlock()
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
func (s *Server) Shutdown() {
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
}

func (s *Server) listener (c *Conn) {
	for {
		b, op, err := wsutil.ReadClientData(c.conn)

		if err != nil {
			s.lock.Lock()
			delete(s.connections, c)
			s.lock.Unlock()
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
		default:
			c.Close()
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

func (s *Server) closeConnection (c *Conn, code ws.StatusCode, reason string) {
	c.CloseWithStatus(code, reason)

	s.lock.Lock()
	delete(s.connections, c)
	s.lock.Unlock()
}

func (s *Server) notFound(c *Conn, msg *Message) {
	c.Emit("not found", msg.Body)
}