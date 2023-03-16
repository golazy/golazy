package lazyroom

import "sync"

type Backend interface {
	Subscribe(channels ...string) error
	Unsubscribe(channels ...string) error
	Emit(e *Event) error
	Events() chan (Event)
}

func NewMemoryBackend() Backend {
	return &memoryBackend{
		subs: make(map[string]struct{}),
		l:    make(chan (Event), 1),
	}
}

type memoryBackend struct {
	sync.RWMutex
	subs map[string]struct{}
	l    chan (Event)
}

func (mb *memoryBackend) Subscribe(channels ...string) error {
	mb.Lock()
	defer mb.Unlock()
	for _, c := range channels {
		mb.subs[c] = struct{}{}
	}
	return nil
}

func (mb *memoryBackend) Unsubscribe(channels ...string) error {
	mb.Lock()
	defer mb.Unlock()
	for _, c := range channels {
		delete(mb.subs, c)
	}
	return nil
}

func (mb *memoryBackend) Emit(e *Event) error {
	mb.RLock()
	defer mb.RUnlock()

	for _, c := range e.Channels {
		if _, ok := mb.subs[c]; ok {
			mb.l <- *e
			return nil
		}
	}

	return nil
}

func (mb *memoryBackend) Events() chan (Event) {
	return mb.l
}
