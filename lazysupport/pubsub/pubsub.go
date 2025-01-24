package pubsub

import "sync"

type Publisher[T any] struct {
	sync.RWMutex
	subs []chan T
}

func (p *Publisher[T]) Subscripe() (<-chan T, func()) {
	p.Lock()
	defer p.Unlock()

	ch := make(chan T, 1)
	p.subs = append(p.subs, ch)

	return ch, func() {
		p.Lock()
		defer p.Unlock()

		for i, c := range p.subs {
			if c == ch {
				p.subs = append(p.subs[:i], p.subs[i+1:]...)
				return
			}
		}
		panic("closing ")
	}
}

func (p *Publisher[T]) Publish(v T) {
	p.RLock()
	defer p.RUnlock()

	for _, c := range p.subs {
		c <- v
	}
}
