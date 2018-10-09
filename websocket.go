// Package websocket implements the WebSocket protocol defined in RFC 6455.
package websocket

import (
	"encoding/json"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"
	"unicode/utf8"
)

// Server allow to keep connection list, broadcast channel and callbacks list
type Server struct {
	connections 	map[*Conn]bool
	channels	 	map[string]*Channel
	broadcast		chan interface{}
	callbacks		map[string]HandlerFunc

	addConn			chan *Conn
	delConn			chan *Conn

	delChan			[]chan *Conn

	onConnect		func (c *Conn)
	onDisconnect	func (c *Conn)

	shutdown		bool
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

// HandleFunc is a type for handle function
// all function which has callback have this struct
// as first element returns pointer to connection
// its give opportunity to close connection or emit message to exactly this connection
type HandlerFunc func (c *Conn, msg *Message)

// Create a new websocket server handler with the provided options.
func Create () *Server {
	return &Server{
		connections: make(map[*Conn]bool),
		channels: make(map[string]*Channel),
		broadcast: make(chan interface{}),
		callbacks: make(map[string]HandlerFunc),
		addConn: make(chan *Conn),
		delConn: make(chan *Conn),
		done: make(chan bool, 1),
		shutdown: false,
	}
}

// CreateAndRun instantly create and run websocket server
func CreateAndRun () *Server {
	s := Create()
	s.Run()
	return s
}

// Run start go routine which listening for channels
func (s *Server) Run () error {
	go func() {
		for {
			select {
			case <- s.done:
				s.shutdown = true
				break
			case <- s.broadcast:
				log.Print("New message in broadcast channel")
			case conn := <- s.addConn:
				if !reflect.ValueOf(s.onConnect).IsNil() {
					go s.onConnect(conn)
				}
				s.connections[conn] = true
			case conn := <- s.delConn:
				if !reflect.ValueOf(s.onDisconnect).IsNil() {
					go s.onDisconnect(conn)
				}
				go func() {
					for _, dC := range s.delChan{
						dC <- conn
					}
				}()
				delete(s.connections, conn)
			}
		}
	}()

	return nil
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping all goroutines
func (s *Server) Shutdown () error {
	l := len(s.connections)
	var wg sync.WaitGroup
	wg.Add(l)

	for c := range s.connections {
		go func(c *Conn) {
			c.Close()
			wg.Done()
		}(c)
	}

	wg.Wait()

	s.done <- true
	time.Sleep(100000 * time.Nanosecond)
	return nil
}

// Handler get upgrade connection to RFC 6455 and starting listener for it
func (s *Server) Handler (w http.ResponseWriter, r *http.Request) {
	if s.shutdown {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("websocket: server not started"))
		return
	}

	con, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("websocket: connection cannot be upgraded %v", err)))
		return
	}
	defer con.Close()

	connection := NewConn(con)
	s.addConn <- connection

	s.listen(connection)
}

// On adding callback for message
func (s *Server) On (name string, f HandlerFunc) {
	s.mu.Lock()
	s.callbacks[name] = f
	s.mu.Unlock()
}

// NewChannel create new channel and proxy channel delConn
// for handling connection closing
func (s *Server) NewChannel (name string) *Channel {
	c := newChannel(name)
	s.delChan = append(s.delChan, c.delConn)
	return c
}

// Set function which will be called when new connections come
func (s *Server) OnConnect (f func(c *Conn)) {
	s.mu.Lock()
	s.onConnect = f
	s.mu.Unlock()
}

// Set function which will be called when new connections come
func (s *Server) OnDisconnect (f func(c *Conn)) {
	s.mu.Lock()
	s.onDisconnect = f
	s.mu.Unlock()
}

// Count return number of active connections.
func (s *Server) Count () int {
	return len(s.connections)
}

func (s *Server) listen (c *Conn) {
	for {
		b, op, err := wsutil.ReadClientData(c.Conn())

		if err != nil {
			s.delConn <- c
			break
		}

		switch op {
		case ws.OpContinuation:
			continue
		case ws.OpText:
			if !utf8.Valid(b) {
				s.delConn <- c
				return
			}
			s.processMessage(c, b)
		case ws.OpBinary:
			s.delConn <- c
		case ws.OpClose:
			s.delConn <- c
		case ws.OpPing:
			c.Pong()
		case ws.OpPong:
			c.Ping()
		}
	}
}

func (s *Server) processMessage (c *Conn, b []byte) error {
	var msg Message
	err := json.Unmarshal(b, &msg)
	if err != nil {
		return err
	}

	if s.callbacks[msg.Name] != nil {
		s.callbacks[msg.Name](c, &msg)
	}

	return nil
}