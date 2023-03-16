package lazyroom

import (
	"fmt"
	"strings"
)

const ChanSep = "/"

type Event struct {
	Channels []string
	Data     []byte
}

func To(c ...any) string {
	a := make([]string, len(c))
	for i, v := range c {
		a[i] = fmt.Sprint(v)
	}
	return strings.Join(a, ChanSep)
}

type epMsg struct {
	c     chan (Event)
	chans []string
	e     *Event
	done  chan (error)
}

type EventProducer struct {
	b            Backend
	listeners    map[string][]chan (Event)
	subscribeC   chan (*epMsg)
	unsubscribeC chan (*epMsg)
	eventC       chan (*epMsg)
}

func NewEventProducer(backend Backend) *EventProducer {
	if backend == nil {
		backend = NewMemoryBackend()
	}

	ep := &EventProducer{
		b: backend,

		listeners: make(map[string][]chan (Event)),

		subscribeC:   make(chan (*epMsg)),
		unsubscribeC: make(chan (*epMsg)),
		eventC:       make(chan (*epMsg)),
	}

	go ep.loop()
	return ep

}

func (ep *EventProducer) loop() {
	for {
		select {
		case msg := <-ep.subscribeC:
			err := ep.b.Subscribe(msg.chans...)
			if err != nil {
				msg.done <- err
				continue
			}
			for _, channel := range msg.chans {
				ep.listeners[channel] = append(ep.listeners[channel], msg.c)
			}

			msg.done <- nil
		case msg := <-ep.unsubscribeC:
			// TODO have an index of channels to listeners
			for name, channels := range ep.listeners {
				for i, channel := range channels {
					if channel == msg.c {
						ep.listeners[name] = append(channels[:i], channels[i+1:]...)
					}
				}
			}
			close(msg.c)

			err := ep.b.Unsubscribe(msg.chans...)
			if err != nil {
				msg.done <- err
				continue
			}
			msg.done <- nil
		case e := <-ep.b.Events():
			for _, channel := range e.Channels {
				for _, c := range ep.listeners[channel] {
					c <- e
				}
			}

		case msg := <-ep.eventC:
			msg.done <- ep.b.Emit(msg.e)
		}
	}
}

func (ep *EventProducer) Subscribe(c chan (Event), channels ...string) error {
	req := &epMsg{c: c, chans: channels, done: make(chan (error))}
	ep.subscribeC <- req
	return <-req.done
}

func (ep *EventProducer) Emit(e *Event) error {
	req := &epMsg{e: e, done: make(chan (error))}
	ep.eventC <- req
	return <-req.done
}

func (ep *EventProducer) Unsubscribe(c chan (Event)) error {
	req := &epMsg{c: c, done: make(chan (error))}
	ep.unsubscribeC <- req
	return <-req.done
}
