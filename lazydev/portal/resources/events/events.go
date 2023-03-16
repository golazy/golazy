package events

import (
	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazyroom"
)

var Events = lazyroom.NewEventProducer(nil)

func Event(e events.Event) {
	Events.Emit(&lazyroom.Event{
		Channels: []string{
			lazyroom.To("devapp", e.Type()),
		},
		Data: []byte(e.String()),
	})
}
