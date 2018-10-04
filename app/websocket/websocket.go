package websocket

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"unicode/utf8"
)

type WebSocket struct {
	connections 	map[*Conn]bool
	broadcast 		chan interface{}
	callbacks 		map[string]HandlerFunc

	lock 			sync.Mutex
}

// main type for handle function
// all function which has callback have this struct
// as first element returns pointer to connection
// its give oportunity to close connection or emit message to exactly this connection
type HandlerFunc func (c *Conn, mes *Message)

// this is a struct for message which sending between application and clients
// Name using for matching callback function in On function
// Body will be transformed to byte array and returned to callback
type Message struct {
	Name 	string 		`json:"name"`
	Body 	string 		`json:"body"`
}

// upgrader from github.com/gorilla/websocket
// its public variable, so if you need another params
// just reassign it before calling New()
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	EnableCompression: true,
}

// main function which initialized main maps
// return *WebSocket struct for next actions
func New() *WebSocket {
	return &WebSocket{
		connections: make(map[*Conn]bool),
		broadcast: make(chan interface{}),
		callbacks: make(map[string]HandlerFunc),
	}
}

// http handler for upgrading connection to RFC 6455 specs
// also adding connection to clients list, and starting listener for it
func (s *WebSocket) Handler (w http.ResponseWriter, r *http.Request) {
	ws, err := Upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("connection cannot be upgraded to websocket"))
		return
	}
	defer ws.Close()

	// TODO: generate normal id for connection
	connection := &Conn{
		id: "test",
		socket: ws,
	}
	s.connections[connection] = true

	s.listener(connection)
}

// just adding callback to list
func (s *WebSocket) On (name string, f HandlerFunc) {
	s.callbacks[name] = f
}

// sending message to all connections
// be aware with it!
// if you need to send message just for one specific user
// call Emit on connection
func (s *WebSocket) Emit () {
	// TODO: try to send message to connections, if connection not alive -> close it
}

// this function must be called before application died
// its goes throw all connection and closing it
func (s *WebSocket) Close () {
	l := len(s.connections)
	var wg sync.WaitGroup
	wg.Add(l)

	for c := range s.connections {
		go func(c *Conn) {
			s.closeConnection(c, websocket.CloseNormalClosure, websocket.ErrCloseSent.Error())
			wg.Done()
		}(c)
	}

	wg.Wait()

	// TODO: check if broadcast is closed
}

// go routines which listening for new message on broadcast channel
// if new message pushed to channel it will be send to connection
func (s *WebSocket) Broadcaster () {
	// TODO: make right broadcaster with closing via context and killing on Close()
}

// this function starts loop which listening for message from clients
// on each request creating own listener
func (s *WebSocket) listener (c *Conn) {
	for {
		mt, b, err := c.socket.ReadMessage()

		if err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				switch ce.Code {
				case websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseAbnormalClosure:
					delete(s.connections, c)
					break
				case websocket.CloseMessage:
					delete(s.connections, c)
					break
				}
			}
			return
		}

		switch mt {
		case websocket.TextMessage:
			if !utf8.Valid(b) {
				s.closeConnection(c, websocket.CloseInvalidFramePayloadData, "not text")
				return
			}
			s.processMessage(c, b)
		case websocket.BinaryMessage:
			s.processMessage(c, b)
		case websocket.CloseMessage:
			s.closeConnection(c, websocket.CloseNormalClosure, websocket.ErrCloseSent.Error())
		case websocket.PingMessage:
			c.Pong()
		case websocket.PongMessage:
			c.Ping()
		}
	}
}

// processing message from clients and calling callback
// if callback with required name not found
// calling default not found function
func (s *WebSocket) processMessage (c *Conn, b []byte) {
	var mes Message
	err := json.Unmarshal(b, &mes)
	if err != nil {
		log.Print("[ERROR] can't unmarshal message")
		// TODO: response via emit and maybe some error?
		return
	}

	if s.callbacks[mes.Name] != nil {
		s.callbacks[mes.Name](c, &mes)
		return
	}

	// TODO: call not found function
}

// calling Close on connection and removed it from connections list
func (s *WebSocket) closeConnection (c *Conn, code int, message string) {
	c.Close(code, message)
	delete(s.connections, c)
}

// TODO: EmitTo function, which emit message to connection via specific id