package websockets

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Message []byte

type Client struct {
	conn *websocket.Conn
}

func (c *Client) SendMessage(m *Message) error {
	return nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) ReceiveMessage() (*Message, error) {
	return nil, nil
}

type WebSockets struct {
}

func (ws *WebSockets) Gen_WebsocketClient(w http.ResponseWriter, r *http.Request) (*Client, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}
