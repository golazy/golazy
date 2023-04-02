package events

import (
	"fmt"

	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazyroom"
	"golazy.dev/lazysupport"
)

var Events = lazyroom.NewEventProducer(nil)

func Subscribe(c chan lazyroom.Event, channel string) {
	Events.Subscribe(c, channel)
}

func Save(e events.Event) {

	var data []byte
	if ed, ok := e.(events.DataEvent); ok {
		data = ed.Data()
	} else {
		data = []byte(e.String())
	}

	chanName := lazyroom.To("devapp", lazysupport.Underscorize(e.Type()))

	fmt.Println("CHAN:", chanName)
	if chanName == "devapp/stdout" {
		fmt.Println("OUT:", string(data))
	}
	if chanName == "devapp/stdout" {
		fmt.Println("OUT:", string(data))
	}
	if chanName == "devapp/stderr" {
		fmt.Println("ERR:", string(data))
	}
	if chanName == "devapp/build_error" {
		fmt.Println("ERR:", string(data))
	}
	if chanName == "devapp/app_stop" {
		fmt.Println("ERR:", string(data))
	}
	Events.Emit(&lazyroom.Event{
		Channels: []string{chanName},
		Data:     data,
	})
}
