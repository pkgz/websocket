// Package websocket implements the WebSocket protocol with additional functions.
/*
Examples

Echo server:
	package main

	import (
		"github.com/pkgz/websocket"
		"net/http"
	)

	func main () {
		r := http.NewServeMux()
		wsServer := websocket.Start(context.Background())

		r.HandleFunc("/ws", wsServer.Handler)

		wsServer.On("echo", func(ctx context.Context, c *websocket.Conn, msg *websocket.Message) {
			c.Emit("echo", msg.Data)
		})

		http.ListenAndServe(":8080", r)
	}

Websocket with group:
	package main

	import (
		"github.com/pkgz/websocket"
		"net/http"
	)

	func main () {
		r := http.NewServeMux()

		wsServer := websocket.Start(context.Background())

		ch := wsServer.NewChannel("test")

		wsServer.OnConnect(func(ctx context.Context, c *websocket.Conn) {
			ch.Add(c)
			ch.Emit("connection", []byte("new connection come"))
		})

		r.HandleFunc("/ws", wsServer.Handler)
		http.ListenAndServe(":8080", r)
	}

*/
package websocket

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sync"
)

// Server allows keeping connection list, broadcast channel and callbacks list.
type Server struct {
	connections map[*Conn]bool
	channels    map[string]*Channel
	broadcast   chan Message
	callbacks   map[string]HandlerFunc

	delChan []chan *Conn

	onConnect    func(ctx context.Context, c *Conn)
	onDisconnect func(ctx context.Context, c *Conn)
	onMessage    func(ctx context.Context, c *Conn, h ws.Header, b []byte)

	done bool
	mu   sync.RWMutex
}

// Message is a struct for data which sending between application and clients.
// Name using for matching callback function in On function.
// Body will be transformed to byte array and returned to callback.
type Message struct {
	Name string `json:"name"`
	Data []byte `json:"data"`
}

// HandlerFunc is a type for handle function all function which has callback have this struct
// as first element returns pointer to connection
// its give opportunity to close connection or emit message to exactly this connection.
type HandlerFunc func(ctx context.Context, c *Conn, msg *Message)

// New websocket server handler with the provided options.
func New() *Server {
	srv := &Server{
		connections: make(map[*Conn]bool),
		channels:    make(map[string]*Channel),
		broadcast:   make(chan Message),
		callbacks:   make(map[string]HandlerFunc),
	}
	srv.onMessage = func(ctx context.Context, c *Conn, h ws.Header, b []byte) {
		_ = c.Write(h, b)
	}
	return srv
}

// Start instantly create and run websocket server.
func Start(ctx context.Context) *Server {
	s := New()
	s.Run(ctx)
	return s
}

// Run start go routine which listening for channels.
func (s *Server) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case msg := <-s.broadcast:
				go func() {
					s.mu.RLock()
					for c := range s.connections {
						_ = c.Emit(msg.Name, msg.Data)
					}
					s.mu.RUnlock()
				}()
			case <-ctx.Done():
				if err := s.Shutdown(); err != nil {
					log.Print(err)
				}
				return
			}
		}
	}()
}

// Shutdown must be called before application died
// its goes throw all connection and closing it
// and stopping all goroutines.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	l := len(s.connections)
	var wg sync.WaitGroup
	wg.Add(l)

	for c := range s.connections {
		go func(c *Conn) {
			if c.conn != nil {
				_ = c.Close()
			}
			wg.Done()
		}(c)
	}

	wg.Wait()

	s.done = true
	return nil
}

// Handler get upgrade connection to RFC 6455 and starting listener for it.
func (s *Server) Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var params url.Values = nil

	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Printf("websocket: upgrade error %v", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	if r.URL.RawQuery != "" {
		params, err = url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			log.Print(err)
			return
		}
	}

	connection := &Conn{
		id:     uuid(),
		params: params,
		conn:   conn,
		done:   make(chan bool, 1),
	}
	connection.startPing()
	s.addConn(ctx, connection)

	textPending := false

	state := ws.StateServerSide
	utf8Reader := wsutil.NewUTF8Reader(nil)
	cipherReader := wsutil.NewCipherReader(nil, [4]byte{0, 0, 0, 0})

	for {
		header, _ := ws.ReadHeader(conn)
		if err = ws.CheckHeader(header, state); err != nil {
			log.Printf("drop ws connection: %v", err)
			s.dropConn(ctx, connection)
			break
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
			_, _ = io.CopyN(io.Discard, conn, header.Length)
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
			if err != nil {
				log.Printf("drop ws connection: OpClose (%v)", err)
			}
			s.dropConn(ctx, connection)
			break
		}

		header.Masked = false
		if err = s.processMessage(ctx, connection, header, payload); err != nil {
			log.Print(err)
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
func (s *Server) NewChannel(id string) *Channel {
	c := newChannel(id)
	s.mu.Lock()
	s.channels[id] = c
	s.delChan = append(s.delChan, c.delConn)
	s.mu.Unlock()
	return c
}

// Channel find and return the channel.
func (s *Server) Channel(id string) *Channel {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.channels[id]
}

// Channels returns channels id with live connections.
func (s *Server) Channels() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := []string{}
	for _, c := range s.channels {
		if c.Count() != 0 {
			list = append(list, c.id)
		}
	}

	return list
}

// OnConnect function which will be called when new connections come.
func (s *Server) OnConnect(f func(ctx context.Context, c *Conn)) {
	s.mu.Lock()
	s.onConnect = f
	s.mu.Unlock()
}

// OnDisconnect function which will be called when new connections come.
func (s *Server) OnDisconnect(f func(ctx context.Context, c *Conn)) {
	s.mu.Lock()
	s.onDisconnect = f
	s.mu.Unlock()
}

// OnMessage handling byte message. This function works as echo by default
func (s *Server) OnMessage(f func(ctx context.Context, c *Conn, h ws.Header, b []byte)) {
	s.mu.Lock()
	s.onMessage = f
	s.mu.Unlock()
}

// Emit message to all connections.
func (s *Server) Emit(name string, data []byte) {
	s.broadcast <- Message{
		Name: name,
		Data: data,
	}
}

// SendTo send message to channel with id.
func (s *Server) SendTo(id string, name string, message *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.Channel(id)
	if ch != nil {
		ch.Emit(name, message)
	}

	return errors.New("no channel found")
}

// Count return number of active connections.
func (s *Server) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections)
}

// IsClosed return the state of websocket server.
func (s *Server) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.done
}

func (s *Server) processMessage(ctx context.Context, c *Conn, h ws.Header, b []byte) error {
	if len(b) == 0 {
		s.onMessage(ctx, c, h, b)
		return nil
	}

	if h.OpCode != ws.OpBinary && h.OpCode != ws.OpText {
		s.onMessage(ctx, c, h, b)
		return nil
	}

	var msg struct {
		Name string `json:"name"`
		Data any    `json:"data"`
	}

	if err := json.Unmarshal(b, &msg); err == nil && s.callbacks[msg.Name] != nil {
		buf, err := json.Marshal(msg.Data)
		if err != nil {
			return err
		}
		s.callbacks[msg.Name](ctx, c, &Message{
			Name: msg.Name,
			Data: buf,
		})
		return nil
	}
	s.onMessage(ctx, c, h, b)

	return nil
}

func (s *Server) addConn(ctx context.Context, conn *Conn) {
	if s.onConnect != nil {
		go s.onConnect(ctx, conn)
	}

	s.mu.Lock()
	s.connections[conn] = true
	s.mu.Unlock()
}

func (s *Server) dropConn(ctx context.Context, conn *Conn) {
	if !reflect.ValueOf(s.onDisconnect).IsNil() {
		go s.onDisconnect(ctx, conn)
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

func uuid() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
