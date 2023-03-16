package lazyroom

import (
	"runtime"
	"strings"
	"testing"
)

func TestEventProducer(t *testing.T) {

	ep := NewEventProducer(nil)

	E := func(s string, data ...string) *Event {
		return &Event{
			Channels: []string{s},
			Data:     []byte(strings.Join(data, " ")),
		}
	}

	events := make(chan (Event), 1000)

	ep.Emit(E("user/1", "msg1"))
	ep.Subscribe(events, "user/1")
	ep.Emit(E("user/1", "msg2"))
	runtime.Gosched() // Allow the backend to process the event
	ep.Unsubscribe(events)
	ep.Emit(E("user/1", "msg3"))

	data := []string{}
	for e := range events {
		data = append(data, string(e.Data))
	}

	result := strings.Join(data, " ")
	if result != "msg2" {
		t.Errorf("Expected 'msg2' got '%s'", result)
	}

}
