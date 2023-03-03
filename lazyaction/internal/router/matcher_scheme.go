package router

import "net/http"

type SchemeMatcher[T any] struct {
	all   Matcher[T]
	http  Matcher[T]
	https Matcher[T]
}

func NewSchemeMatcher[T any]() Matcher[T] {
	return &SchemeMatcher[T]{
		all:   NewDomainMatcher[T](),
		http:  NewDomainMatcher[T](),
		https: NewDomainMatcher[T](),
	}
}

func (sm *SchemeMatcher[T]) All() []Route[T] {
	all := []Route[T]{}

	for _, r := range sm.all.All() {
		r.Req.URL.Scheme = ""
		all = append(all, r)
	}

	for _, r := range sm.http.All() {
		r.Req.URL.Scheme = "http"
		all = append(all, r)
	}

	for _, r := range sm.https.All() {
		r.Req.URL.Scheme = "https"
		all = append(all, r)
	}

	return all

}
func (sm *SchemeMatcher[T]) Add(req *RouteDefinition, t *T) {
	switch req.Scheme {
	case "http":

		sm.http.Add(req, t)
	case "https":
		sm.https.Add(req, t)
	default:
		sm.all.Add(req, t)
	}
}

func (sm *SchemeMatcher[T]) Find(req *http.Request) *T {
	switch req.URL.Scheme {
	case "http":
		if t := sm.http.Find(req); t != nil {
			return t
		}
	case "https":
		if t := sm.https.Find(req); t != nil {
			return t
		}
	}
	t := sm.all.Find(req)
	if t != nil {
		return t
	}
	// Even for production we will allways have scheme, for tests we can have
	// empty scheme, so we will try to find it in all routes
	t = sm.http.Find(req)
	if t != nil {
		return t
	}
	return sm.https.Find(req)
}
