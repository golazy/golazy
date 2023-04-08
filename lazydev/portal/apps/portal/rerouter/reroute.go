package rerouter

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"portal/resources/events"
	"strings"
	"sync"

	"golazy.dev/lazyroom"
)

type rerouter struct {
	sync.RWMutex
	h http.Handler
	c chan (lazyroom.Event)
	u *url.URL
}

func (r *rerouter) loop() {
	for e := range r.c {
		for _, ch := range e.Channels {
			switch ch {
			case "devapp/app_start":
				r.Lock()
				u, err := url.Parse(string(e.Data))
				if err != nil {
					panic(err)
				}
				fmt.Println("Rerouting traffic to", u)
				r.u = u
				r.Unlock()
			case "devapp/app_stop":
				r.Lock()
				fmt.Println("Disabling rerouting")
				r.u = nil
				r.Unlock()
			}
		}
	}

}
func (r *rerouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/golazy/") {
		r.h.ServeHTTP(w, req)
		return
	}
	r.RLock()
	u := r.u
	r.RUnlock()

	if u == nil {
		fmt.Println("Got request to portal")

		q := req.URL.Query()
		q.Add("original", req.URL.Path)
		req.URL.RawQuery = q.Encode()
		req.URL.Path = "/golazy/builds/rerouter"
		r.h.ServeHTTP(w, req)
		return
	}

	httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, req)
}

func New(h http.Handler) http.Handler {

	c := make(chan (lazyroom.Event))
	err := events.Events.Subscribe(c, "devapp/app_start", "devapp/app_stop")
	if err != nil {
		panic(err)
	}

	rr := &rerouter{
		h: h,
		c: c,
	}

	go rr.loop()

	return rr

}
