// Package websocket implements the WebSocket protocol with additional functions.
/*
Examples

Echo server:
	package main

	import (
		"github.com/exelban/websocket"
		"github.com/go-chi/chi"
		"net/http"
	)

	func main () {
		r := chi.NewRouter()
		wsServer := websocket.CreateAndRun()

		r.Get("/ws", wsServer.Handler)
		wsServer.On("echo", func(c *websocket.Conn, msg *websocket.Message) {
			c.Emit("echo", msg.Body)
		})

		http.ListenAndServe(":8080", r)
	}

Websocket with group:
	package main

	import (
		"github.com/exelban/websocket"
		"github.com/go-chi/chi"
		"net/http"
	)

	func main () {
		r := chi.NewRouter()
		wsServer := websocket.CreateAndRun()

		ch := wsServer.NewChannel("test")

		wsServer.OnConnect(func(c *websocket.Conn) {
			ch.Add(c)
			ch.Emit("connection", []byte("new connection come"))
		})

		r.Get("/ws", wsServer.Handler)
		http.ListenAndServe(":8080", r)
	}

*/
package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"
)

// Server allow to keep connection list, broadcast channel and callbacks list.
type Server struct {
	connections map[*Conn]bool
	channels    map[string]*Channel
	broadcast   chan *Message
	callbacks   map[string]HandlerFunc

	addConn chan *Conn
	delConn chan *Conn

	delChan []chan *Conn

	onConnect    func(c *Conn)
	onDisconnect func(c *Conn)
	onMessage    func(c *Conn, h ws.Header, b []byte)

	shutdown bool
	done     chan bool

	mu sync.RWMutex
}

// Message is a struct for data which sending between application and clients.
// Name using for matching callback function in On function.
// Body will be transformed to byte array and returned to callback.
type Message struct {
	Name string      `json:"name"`
	Body interface{} `json:"body"`
}

// HandleFunc is a type for handle function
// all function which has callback have this struct
// as first element returns pointer to connection
// its give opportunity to close connection or emit message to exactly this connection.
type HandlerFunc func(c *Conn, msg *Message)

// Create a new websocket server handler with the provided options.
func Create() *Server {
	srv := &Server{
		connections: make(map[*Conn]bool),
		channels:    make(map[string]*Channel),
		broadcast:   make(chan *Message),
		callbacks:   make(map[string]HandlerFunc),
		addConn:     make(chan *Conn),
		delConn:     make(chan *Conn),
		done:        make(chan bool, 1),
		shutdown:    false,
	}
	srv.onMessage = func(c *Conn, h ws.Header, b []byte) {
		_ = c.Write(h, b)
	}
	return srv
}

// CreateAndRun instantly create and run websocket server.
func CreateAndRun() *Server {
	s := Create()
	s.Run()
	return s
}

// Run start go routine which listening for channels.
func (s *Server) Run() {
	go func() {
		for {
			select {
			case <-s.done:
				break
			case msg := <-s.broadcast:
				go func() {
					s.mu.RLock()
					for c := range s.connections {
						_ = c.Emit(msg.Name, msg.Body)
					}
					s.mu.RUnlock()
				}()
			case conn := <-s.addConn:
				if !reflect.ValueOf(s.onConnect).IsNil() {
					go s.onConnect(conn)
				}
				s.mu.Lock()
				s.connections[conn] = true
				s.mu.Unlock()
			case conn := <-s.delConn:
				if !reflect.ValueOf(s.onDisconnect).IsNil() {
					go s.onDisconnect(conn)
				}
				go func() {
					for _, dC := range s.delChan {
						dC <- conn
					}
				}()
				s.mu.Lock()
				delete(s.connections, conn)
				s.mu.Unlock()
			}
		}
	}()
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping all goroutines.
func (s *Server) Shutdown() error {
	s.shutdown = true

	s.mu.RLock()
	defer s.mu.RUnlock()

	l := len(s.connections)
	var wg sync.WaitGroup
	wg.Add(l)

	for c := range s.connections {
		go func(c *Conn) {
			_ = c.Close()
			wg.Done()
		}(c)
	}

	wg.Wait()

	s.done <- true
	time.Sleep(100000 * time.Nanosecond)
	return nil
}

// Handler get upgrade connection to RFC 6455 and starting listener for it.
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	if s.shutdown {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("websocket: server not started"))
		return
	}

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Printf("websocket: upgrade error %v", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	connection := &Conn{
		conn: conn,
	}
	s.addConn <- connection

	textPending := false

	state := ws.StateServerSide
	utf8Reader := wsutil.NewUTF8Reader(nil)
	cipherReader := wsutil.NewCipherReader(nil, [4]byte{0, 0, 0, 0})

	for {
		header, err := ws.ReadHeader(conn)
		if err = ws.CheckHeader(header, state); err != nil {
			s.delConn <- connection
			return
		}

		cipherReader.Reset(io.LimitReader(conn, header.Length), header.Mask)

		var utf8Fin bool
		var r io.Reader = cipherReader

		switch header.OpCode {
		case ws.OpPing:
			header.OpCode = ws.OpPong
			header.Masked = false
			_ = ws.WriteHeader(conn, header)
			_, _ = io.CopyN(conn, cipherReader, header.Length)
			continue
		case ws.OpPong:
			_, _ = io.CopyN(ioutil.Discard, conn, header.Length)
			continue
		case ws.OpClose:
			utf8Fin = true
		case ws.OpContinuation:
			if textPending {
				utf8Reader.Source = cipherReader
				r = utf8Reader
			}
			if header.Fin {
				state = state.Clear(ws.StateFragmented)
				textPending = false
				utf8Fin = true
			}
		case ws.OpText:
			utf8Reader.Reset(cipherReader)
			r = utf8Reader

			if !header.Fin {
				state = state.Set(ws.StateFragmented)
				textPending = true
			} else {
				utf8Fin = true
			}
		case ws.OpBinary:
			if !header.Fin {
				state = state.Set(ws.StateFragmented)
			}
		}

		payload := make([]byte, header.Length)
		_, err = io.ReadFull(r, payload)
		if err == nil && utf8Fin && !utf8Reader.Valid() {
			err = wsutil.ErrInvalidUTF8
		}

		if err != nil || header.OpCode == ws.OpClose {
			s.delConn <- connection
			return
		}

		header.Masked = false
		err = s.processMessage(connection, header, payload)
		if err != nil {
			log.Print(err)
			s.delConn <- connection
			return
		}
	}
}

// On adding callback for message.
func (s *Server) On(name string, f HandlerFunc) {
	s.mu.Lock()
	s.callbacks[name] = f
	s.mu.Unlock()
}

// NewChannel create new channel and proxy channel delConn
// for handling connection closing.
func (s *Server) NewChannel(name string) *Channel {
	c := newChannel(name)
	s.mu.Lock()
	s.delChan = append(s.delChan, c.delConn)
	s.mu.Unlock()
	return c
}

// Set function which will be called when new connections come.
func (s *Server) OnConnect(f func(c *Conn)) {
	s.mu.Lock()
	s.onConnect = f
	s.mu.Unlock()
}

// Set function which will be called when new connections come.
func (s *Server) OnDisconnect(f func(c *Conn)) {
	s.mu.Lock()
	s.onDisconnect = f
	s.mu.Unlock()
}

// OnMessage handling byte message. By default this function works as echo.
func (s *Server) OnMessage(f func(c *Conn, h ws.Header, b []byte)) {
	s.mu.Lock()
	s.onMessage = f
	s.mu.Unlock()
}

// Emit emit message to all connections.
func (s *Server) Emit(name string, body interface{}) {
	msg := Message{
		Name: name,
		Body: body,
	}
	s.broadcast <- &msg
}

// Count return number of active connections.
func (s *Server) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

func (s *Server) processMessage(c *Conn, h ws.Header, b []byte) error {
	var msg Message

	if len(b) == 0 {
		s.onMessage(c, h, b)
		return nil
	}

	switch h.OpCode {
	case ws.OpBinary:
		s.onMessage(c, h, b)
	case ws.OpText:
		switch b[0] {
		case 123:
			err := json.Unmarshal(b, &msg)
			if err != nil {
				return err
			}
			if s.callbacks[msg.Name] != nil {
				s.callbacks[msg.Name](c, &msg)
			}
		default:
			s.onMessage(c, h, b)
			return nil
		}
	default:
		s.onMessage(c, h, b)
	}

	return nil
}
