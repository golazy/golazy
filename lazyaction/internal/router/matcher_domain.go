package router

import (
	"net/http"
	"regexp"
	"strings"
)

type domain[T any] struct {
	domain string
	routes Matcher[T]
	re     *regexp.Regexp
}

func newDomain[T any](tld string) domain[T] {

	d := domain[T]{
		domain: tld,
		routes: NewPortMatcher[T](),
	}
	if strings.ContainsAny(tld, "*,()") {
		reStr := strings.ReplaceAll(tld, ".", `\.`)
		reStr = strings.ReplaceAll(reStr, "*", `.*`)
		reStr = strings.ReplaceAll(reStr, ",", `|`)
		d.re = regexp.MustCompile("^" + reStr + "$")
	}

	return d

}

func (d domain[T]) Match(domain string) bool {
	if d.re != nil {
		return d.re.MatchString(domain)
	}
	return d.domain == domain
}

type DomainMatcher[T any] struct {
	any     Matcher[T]
	domains []domain[T]
}

func NewDomainMatcher[T any]() Matcher[T] {
	return &DomainMatcher[T]{
		any:     NewPortMatcher[T](),
		domains: []domain[T]{},
	}
}

func (r *DomainMatcher[T]) All() []Route[T] {
	all := []Route[T]{}

	for _, d := range r.domains {
		for _, path := range d.routes.All() {
			port := path.Req.URL.Port()
			path.Req.URL.Host = d.domain
			if port != "" {
				path.Req.URL.Host += ":" + port
			}
			all = append(all, path)
		}
	}

	for _, path := range r.any.All() {
		port := path.Req.URL.Port()
		path.Req.URL.Host = ""
		if port != "" {
			path.Req.URL.Host += ":" + port
		}
		all = append(all, path)
	}
	return all

}
func (r *DomainMatcher[T]) Add(req *http.Request, t *T) {
	domain := req.URL.Hostname()

	if domain == "*" {
		domain = ""
	}
	if domain == "" {
		r.any.Add(req, t)
		return
	}

	for _, d := range r.domains {
		if d.domain == domain {
			d.routes.Add(req, t)
			return
		}
	}

	d := newDomain[T](domain)

	d.routes.Add(req, t)
	r.domains = append(r.domains, d)
}

func (r DomainMatcher[T]) Find(req *http.Request) *T {
	domain := req.URL.Hostname()

	for _, d := range r.domains {
		if d.Match(domain) {
			target := d.routes.Find(req)
			if target != nil {
				return target
			}
		}
	}

	return r.any.Find(req)
}
