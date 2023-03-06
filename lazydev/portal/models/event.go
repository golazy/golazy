package models

import (
	"fmt"
	"sync"

	"golazy.dev/lazydev/devserver/events"
)

var MaxEvents = 100

type eventSub struct {
	E     chan events.Event
	types []string
}

var eventsSubs []*eventSub
var eventsSubsL sync.Mutex
var eventsDB []events.Event
var eventsDBLock sync.Mutex

func EventSave(e events.Event) {
	fmt.Println(e.Type())
	eventsDBLock.Lock()
	eventsDB = append(eventsDB, e)
	if len(eventsDB) > MaxEvents {
		eventsDB = eventsDB[1:]
	}
	eventsDBLock.Unlock()

	eventsSubsL.Lock()
	for _, sub := range eventsSubs {
		if len(sub.types) == 0 {
			sub.E <- e
			continue
		}
		for _, t := range sub.types {
			if t == e.Type() {
				sub.E <- e
			}
		}
	}
	eventsSubsL.Unlock()

	event_process(e)
}

func EventEach(fn func(events.Event)) {
	eventsDBLock.Lock()
	for _, e := range eventsDB {
		fn(e)
	}
	eventsDBLock.Unlock()
}

func EventSubscribe(event_type ...string) <-chan events.Event {
	ch := make(chan events.Event, 100)
	es := &eventSub{E: ch, types: event_type}
	eventsSubsL.Lock()
	eventsSubs = append(eventsSubs, es)
	eventsSubsL.Unlock()
	return es.E
}

func EventUnsubscribe(ch <-chan events.Event) {
	eventsSubsL.Lock()
	defer eventsSubsL.Unlock()
	for i, sub := range eventsSubs {
		if sub.E == ch {
			eventsSubs = append(eventsSubs[:i], eventsSubs[i+1:]...)
			close(sub.E)
			return
		}
	}
	panic("trying to unsubscribe from non subscribed channel")
}

func EventAll() []events.Event {
	eventsDBLock.Lock()
	e := make([]events.Event, len(eventsDB))
	copy(e, eventsDB)
	defer eventsDBLock.Unlock()
	return e
}

func event_process(e events.Event) {
	fmt.Println(e)

	switch e := e.(type) {
	case events.Listen:
	case events.BuildStart:
	case events.BuildSuccess:
		BuildUpdate(true, nil)
	case events.BuildError:
		BuildUpdate(false, e.Out)
	case events.AppStart:
		AppSetURL(e.URL)
	case events.AppStop:
		AppSetURL(nil)
	case events.AppStartError:
	case events.FSChange:
	}
}
