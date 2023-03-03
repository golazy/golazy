package server_test

import (
	"net/http"
	"net/url"
	"sync"

	"golazy.dev/lazydev/server"
)

type TestDevApp struct {
	E        chan (server.Event)
	NoPortal bool
	url      *url.URL
	events   []server.Event
	l        sync.Mutex
}

func (a *TestDevApp) waitFor(ev string) {
	for e := range a.E {
		if e.Type() == ev {
			return
		}
	}
}

func (a *TestDevApp) Event(e server.Event) {
	a.l.Lock()
	defer a.l.Unlock()

	a.E <- e

	a.events = append(a.events, e)
	switch e := e.(type) {
	case server.EventAppStart:
		a.url = e.URL
	case server.EventBuildError:
		a.url = nil
	case server.EventAppStop:
		a.url = nil
	}
}

func (a *TestDevApp) DisablePortal() {
	a.NoPortal = true
}

func (a *TestDevApp) Open(url *url.URL) {
	a.url = url
}

func (a *TestDevApp) Close() {
	a.url = nil
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
