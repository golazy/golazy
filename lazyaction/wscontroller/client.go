package wscontroller

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/websocket"
)

type Client struct {
	c *websocket.Conn
	r *http.Request
}

func (c *Client) Send(msg any) error {
	switch msg := msg.(type) {
	case []byte:
		return c.c.WriteMessage(websocket.BinaryMessage, msg)
	case string:
		return c.c.WriteMessage(websocket.TextMessage, []byte(msg))
	default:
		return c.c.WriteJSON(msg)
	}

}

func (c *Client) SendCommand(cmd string, obj any) error {

	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	return c.c.WriteJSON(WSMessage{
		Command: cmd,
		Data:    string(data),
	})
}
