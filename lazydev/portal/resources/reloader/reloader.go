package reloader

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Reloader struct {
	WebSocket
}

func (rel *Reloader) Index(c *Client) {

}

func (re *Reloader) WS() {}

type WebSocket struct {
	sync.Mutex
	clients []*Client
	close   func()
}

func (ws *WebSocket) SaveClient(c *Client) {
	ws.Lock()
	ws.clients = append(ws.clients, c)
	ws.Unlock()
}

func (ws *WebSocket) GenWS(w http.ResponseWriter, r *http.Request) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	c := &Client{
		conn: conn,
		r:    r,
		close: func() {
			ws.Lock()
			for i, c := range ws.clients {
				if c == conn {
					ws.clients = append(ws.clients[:i], ws.clients[i+1:]...)
					break
				}
			}
			ws.Unlock()
		},
	}

	ws.Lock()
	ws.clients = append(ws.clients, c)
	ws.Unlock()

	return c, nil
}

type Client struct {
	conn *websocket.Conn
	r    *http.Request
	tags []string
	ids  []uint64
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
