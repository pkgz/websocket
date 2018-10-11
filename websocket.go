// Package websocket implements the WebSocket protocol with additional functions.
package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
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

	mu sync.Mutex
}

// Message is a struct for data which sending between application and clients.
// Name using for matching callback function in On function.
// Body will be transformed to byte array and returned to callback.
type Message struct {
	Name string `json:"name"`
	Body []byte `json:"body"`
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
		c.Write(h, b)
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
				s.shutdown = true
				break
			case msg := <-s.broadcast:
				go func() {
					for c := range s.connections {
						c.Emit(msg.Name, msg.Body)
					}
				}()
			case conn := <-s.addConn:
				if !reflect.ValueOf(s.onConnect).IsNil() {
					go s.onConnect(conn)
				}
				s.connections[conn] = true
			case conn := <-s.delConn:
				if !reflect.ValueOf(s.onDisconnect).IsNil() {
					go s.onDisconnect(conn)
				}
				go func() {
					for _, dC := range s.delChan {
						dC <- conn
					}
				}()
				delete(s.connections, conn)
			}
		}
	}()
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping all goroutines.
func (s *Server) Shutdown() error {
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

// Handler get upgrade connection to RFC 6455 and starting listener for it.
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	if s.shutdown {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("websocket: server not started"))
		return
	}

	conn, _, _, _ := ws.UpgradeHTTP(r, w)
	defer conn.Close()

	connection := NewConn(conn)
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
			b := make([]byte, header.Length)
			_, err = io.ReadFull(r, b)
			connection.Pong(b)
			continue
		case ws.OpPong:
			b := make([]byte, header.Length)
			_, err = io.ReadFull(r, b)
			connection.Ping(b)
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
func (s *Server) Emit(name string, body []byte) {
	msg := Message{
		Name: name,
		Body: body,
	}
	s.broadcast <- &msg
}

// Count return number of active connections.
func (s *Server) Count() int {
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
	}

	return nil
}
