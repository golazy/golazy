package wscontroller

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSHandler interface {
	http.Handler
	Each(func(c *Client, controller any))
}

func NewWSHandler(obj any) WSHandler {

	return newWrapper(obj)

}
