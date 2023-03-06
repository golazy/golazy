package portalserver_test

import (
	"net/http"
	"net/url"
	"sync"

	"golazy.dev/lazydev/devserver/events"
)

type TestDevApp struct {
	E        chan (events.Event)
	s        *http.Server
	NoPortal bool
	url      *url.URL
	events   []events.Event
	l        sync.Mutex
}

func (a *TestDevApp) waitFor(ev string) events.Event {
	for e := range a.E {
		if e.Type() == ev {
			return e
		}
	}
	return nil
}

func (a *TestDevApp) Event(e events.Event) {
	a.l.Lock()
	defer a.l.Unlock()

	a.E <- e

	a.events = append(a.events, e)
	switch e := e.(type) {
	case events.AppStart:
		a.url = e.URL
	case events.BuildError:
		a.url = nil
	case events.AppStop:
		a.url = nil
	}
}

func (a *TestDevApp) Close() error {
	return a.s.Close()
}

func (a *TestDevApp) ListenAndServe(addr string) error {
	s := &http.Server{
		Addr:    addr,
		Handler: a,
	}
	return s.ListenAndServe()
}

func (a *TestDevApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.url != nil {
		w.Write([]byte("proxy"))
		return
	}
	if a.NoPortal {
		w.Write([]byte("noportal"))
		return
	}

	w.Write([]byte("portal"))
}
