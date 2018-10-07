package websocket

import (
	"encoding/json"
	"flag"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"testing"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

func TestCreate(t *testing.T) {
	ws := Create()

	require.Equal(t, len(ws.connections), 0, "connection list must be empty")
	require.Equal(t, len(ws.broadcast), 0, "broadcast channel must be empty")
	require.Equal(t, len(ws.callbacks), 0, "callbacks list must be empty")
	require.Equal(t, len(ws.done), 0, "done channel must be empty")
}

func TestCreateAndRun(t *testing.T) {
	ws := wsServer()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	require.Equal(t, len(ws.connections), 1, "must be 1 connection active")
}

func TestServer_Handler(t *testing.T) {
	// TODO: send right upgrader massage and one wring
}

func TestServer_Broadcast(t *testing.T) {
	wsServer()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	msg := Message{
		Name: "echo",
		Body: "Hello World",
	}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	err = c.WriteMessage(1, b)
	if err != nil {
		t.Fatal(err)
	}

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			t.Fatal(err)
			break
		}
		require.Equal(t, mt, 1, "must be text message type")
		require.Equal(t, message, b, "response must be the same as request")
		break
	}
}

func TestServer_Emit(t *testing.T) {
	wsServer()
	done := make(chan bool, 2)

	msg := Message{
		Name: "echo",
		Body: "Hello World WoW",
	}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal("dial:", err)
		}
		defer c.Close()

		err = c.WriteMessage(1, b)
		if err != nil {
			t.Fatal(err)
		}

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				t.Fatal(err)
				break
			}
			require.Equal(t, mt, 1, "must be text message type")
			require.Equal(t, message, b, "response must be the same as request")
			done <- true
			break
		}
	}()

	go func() {
		u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
		c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
		if err != nil {
			t.Fatal("dial:", err)
		}
		defer c.Close()

		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				t.Fatal(err)
				break
			}
			require.Equal(t, mt, 1, "must be text message type")
			require.Equal(t, message, b, "response must be the same as request")
			done <- true
			break
		}
	}()

	d1 := <- done
	d2 := <- done

	require.Equal(t, d1, true)
	require.Equal(t, d2, true)
}

func TestServer_On(t *testing.T) {
	wsServer()

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer c.Close()

	msg1 := Message{
		Name: "TestOnFunc",
		Body: "test 1",
	}
	b1, err := json.Marshal(msg1)
	if err != nil {
		t.Fatal(err)
	}

	msg2 := Message{
		Name: "TestOnFunc_wrong",
		Body: "test 2",
	}
	b2, err := json.Marshal(msg2)
	if err != nil {
		t.Fatal(err)
	}

	err = c.WriteMessage(1, b1)
	if err != nil {
		t.Fatal(err)
	}
	err = c.WriteMessage(1, b2)
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			t.Fatal(err)
			break
		}
		i++

		switch i {
		case 1:
			require.Equal(t, message, b1, "response must be the same as request 1")
			continue
		case 2:
			require.NotEqual(t, message, b2, "response must be the same as request 2")
			msg2Check := Message{
				Name: "not found",
				Body: msg2.Body,
			}
			b2Check, err := json.Marshal(msg2Check)
			if err != nil {
				t.Fatal(err)
			}
			require.Equal(t, message, b2Check, "response must be the same as request 2")
		}
		break
	}
}

func TestServer_OnConnect(t *testing.T) {
	// TODO: check onConnect function
}

func TestServer_Shutdown(t *testing.T) {
	ws := wsServer()

	require.Equal(t, len(ws.done), 0, "done channel must be empty")
	ws.Shutdown()
	require.Equal(t, len(ws.done), 1, "done channel cannot be empty")
}

//func TestConn_Ping(t *testing.T) {
//	wsServer()
//
//	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
//	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
//	if err != nil {
//		t.Fatal("dial:", err)
//	}
//	defer c.Close()
//
//	err = c.WriteControl(9, []byte("ping"), time.Now().Add(5*time.Second))
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	for {
//		mt, message, err := c.ReadMessage()
//		if err != nil {
//			t.Fatal(err)
//			break
//		}
//		log.Print(mt, message)
//	}
//}

func wsServer () *Server {
	r := chi.NewRouter()
	ws := CreateAndRun()

	r.Get("/ws", ws.Handler)

	ws.On("echo", func(c *Conn, msg *Message) {
		ws.Emit(msg.Name, msg.Body)
	})

	ws.On("TestOnFunc", func(c *Conn, msg *Message) {
		c.Emit(msg.Name, msg.Body)
	})

	go func() {
		srv := &http.Server{
			Addr: *addr,
			Handler: r,
		}
		srv.ListenAndServe()
	}()

	return &ws
}