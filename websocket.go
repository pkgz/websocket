// Package websocket implements the WebSocket protocol defined in RFC 6455.
package websocket

import (
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"net/http"
	"reflect"
	"sync"
	"time"
)

// Server allow to keep connection list, broadcast channel and callbacks list.
type Server struct {
	connections 	map[*Conn]bool
	channels	 	map[string]*Channel
	broadcast		chan *Message
	callbacks		map[string]HandlerFunc

	addConn			chan *Conn
	delConn			chan *Conn

	delChan			[]chan *Conn

	onConnect		func (c *Conn)
	onDisconnect	func (c *Conn)

	shutdown		bool
	done 			chan bool
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
type HandlerFunc func (c *Conn, msg *Message)

// Create a new websocket server handler with the provided options.
func Create () *Server {
	return &Server{
		connections: make(map[*Conn]bool),
		channels: make(map[string]*Channel),
		broadcast: make(chan *Message),
		callbacks: make(map[string]HandlerFunc),
		addConn: make(chan *Conn),
		delConn: make(chan *Conn),
		done: make(chan bool, 1),
		shutdown: false,
	}
}

// CreateAndRun instantly create and run websocket server.
func CreateAndRun () *Server {
	s := Create()
	s.Run()
	return s
}

// Run start go routine which listening for channels.
func (s *Server) Run () {
	go func() {
		for {
			select {
			case <- s.done:
				s.shutdown = true
				break
			case msg := <- s.broadcast:
				go func() {
					for c := range s.connections{
						c.Emit(msg.Name, msg.Body)
					}
				}()
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
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping all goroutines.
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

// Handler get upgrade connection to RFC 6455 and starting listener for it.
func (s *Server) Handler (w http.ResponseWriter, r *http.Request) {
	if s.shutdown {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("websocket: server not started"))
		return
	}

	con, _, _, _ := ws.UpgradeHTTP(r, w)
	defer con.Close()

	connection := NewConn(con)
	s.addConn <- connection

	state := ws.StateServerSide

	textPending := false
	utf8Reader := wsutil.NewUTF8Reader(nil)
	cipherReader := wsutil.NewCipherReader(nil, [4]byte{0, 0, 0, 0})

	for {
		header, err := ws.ReadHeader(con)
		if err = ws.CheckHeader(header, state); err != nil {
			s.delConn <- connection
			return
		}

		cipherReader.Reset(io.LimitReader(con, header.Length), header.Mask, )

		var utf8Fin bool
		var r io.Reader = cipherReader

		switch header.OpCode {
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
		case ws.OpPing:
			connection.Pong()
			continue
		case ws.OpPong:
			connection.Ping()
			continue
		case ws.OpClose:
			s.delConn <- connection
			return
		}

		payload := make([]byte, header.Length)
		_, err = io.ReadFull(r, payload)
		if err == nil && utf8Fin && !utf8Reader.Valid() {
			err = wsutil.ErrInvalidUTF8
		}

		if err != nil {
			s.delConn <- connection
			return
		}

		s.processMessage(connection, payload)
	}
}

// On adding callback for message.
func (s *Server) On (name string, f HandlerFunc) {
	s.callbacks[name] = f
}

// NewChannel create new channel and proxy channel delConn
// for handling connection closing.
func (s *Server) NewChannel (name string) *Channel {
	c := newChannel(name)
	s.delChan = append(s.delChan, c.delConn)
	return c
}

// Set function which will be called when new connections come.
func (s *Server) OnConnect (f func(c *Conn)) {
	s.onConnect = f
}

// Set function which will be called when new connections come.
func (s *Server) OnDisconnect (f func(c *Conn)) {
	s.onDisconnect = f
}

// Emit emit message to all connections.
func (s *Server) Emit (name string, body []byte) {
	msg := Message{
		Name: name,
		Body: body,
	}
	s.broadcast <- &msg
}

// Count return number of active connections.
func (s *Server) Count () int {
	return len(s.connections)
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