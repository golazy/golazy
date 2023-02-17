package lazydev

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

func newCommander() *commander {
	return &commander{}
}

type commander struct {
	sync.RWMutex
	subs []subscription
}

type subscription chan (msg)

func (c *commander) Broadcast(m msg) {
	c.RLock()
	defer c.RUnlock()

	for _, s := range c.subs {
		s <- m
	}

}

type msg struct {
	Command string
}

func (c *commander) Subscribe() subscription {
	fmt.Println("Got a new connection")
	c.Lock()
	defer c.Unlock()

	sub := subscription(make(chan (msg)))
	c.subs = append(c.subs, sub)
	return sub
}

func (c *commander) Unsubscribe(s subscription) {
	fmt.Println("Got an unsubscribe")
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
	control = newCommander()
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			control.Broadcast(msg{"tick"})
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
			fmt.Println("Channel closed")
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
