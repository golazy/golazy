package devapp

import (
	"portal/layouts/golazy"
)

type Controller struct {
	golazy.Layout
}

func (a *Controller) Index() string {
	return "hellos"

	/*
		u := models.AppURL()
		if u == nil {
			models.EventEach(func(e events.Event) {
				w.Write([]byte(lazyapp.Current.Name))
				w.Write([]byte(e.String() + "\n"))
			})
			w.Write([]byte("Not running"))
			return ""
		}

		httputil.NewSingleHostReverseProxy(u).ServeHTTP(w, r)
		return "asdf"
	*/
}

func (a *Controller) Status() string {
	return "ok"
}
