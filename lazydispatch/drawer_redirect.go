package lazydispatch

import (
	"net/http"
)

type redirect struct {
	scope *Scope
	to    string
	code  int
}

func (s *Scope) RedirectTo(path string, code ...int) {
	r := &redirect{
		scope: s,
		to:    path,
	}
	if len(code) > 0 {
		r.code = code[0]
	} else {
		r.code = 301 // Permanent redirect
	}
	s.addrgen(r)
}

func (redirect *redirect) routes() []*Route {

	r := &Route{}
	r.Method, r.URL, r.Name, _, r.Models = redirect.scope.routeInfo()
	r.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirect.to, redirect.code)
	})

	return []*Route{r}
}
