package commander

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func New() *commander {
	return &commander{}
}

type commander struct {
	sync.RWMutex
	subs []subscription
}

type subscription chan (Msg)

func (c *commander) Broadcast(m Msg) {
	c.RLock()
	defer c.RUnlock()

	for _, s := range c.subs {
		s <- m
	}

}

type Msg struct {
	Command string
}

func (c *commander) Subscribe() subscription {
	c.Lock()
	defer c.Unlock()

	sub := subscription(make(chan (Msg)))
	c.subs = append(c.subs, sub)
	return sub
}

func (c *commander) Unsubscribe(s subscription) {
	c.Lock()
	defer c.Unlock()

	for i, sub := range c.subs {
		if s == sub {
			c.subs = append(c.subs[:i], c.subs[i+1:]...)
			return
		}
	}
}

var control *commander

func init() {
	control = New()
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			control.Broadcast(Msg{"tick"})
		}
	}()
}

func (c *commander) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	sub := c.Subscribe()
	for {
		m, ok := <-sub
		if !ok {
			fmt.Println("Channsl closed")
			return
		}

		fmt.Println("Fowarding a message", m)
		err := conn.WriteJSON(m)
		if err != nil {
			fmt.Println("Error Writing to the client", err)
			c.Unsubscribe(sub)
		}
	}
}
