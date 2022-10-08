/*
Package router implements a http router that support url params and http verbs

It was design to use together lazyaction/controller:

But it can be used alone:

	say := func(what string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(fmt.Sprint(what, r.Form)))
		}
	}

	router := &Router{
		PrefixCatchAll{"page_id", Routes{
			RoutePath{"", "", say("show_page")},
			RoutePath{"publish", "", say("publish_page")},
		}},
		RoutePath{"posts", "", say("posts_index")},
		Prefix{"posts", Routes{
			RouteCatchAll{"post_id", "", say("post_show")},
			RoutePath{"", "", say("post index")},
			RoutePath{"new", "", say("post new")},
			PrefixCatchAll{"post_id", Routes{
				RoutePath{"publish", "", say("publish")},
			}},
			RoutePath{"publish", "", say("publish")},
		}},
	}
*/
package router

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}

type HandlerFunc func(w ResponseWriter, r *Request)

func (h HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	h(w, r)
}

type Method string

const (
	GET    Method = "GET"
	PUT    Method = "PUT"
	POST   Method = "POST"
	DELETE Method = "DELETE"
	PATCH  Method = "PATCH"
)

type Router Routes

type PathSegment interface{}

type Routes []PathSegment

type CatchAllPath struct {
	ParamName   string
	Method      string
	Destination string
	Handler
}

type Prefix struct {
	Prefix string
	Routes Routes
}

type CatchAllPrefix struct {
	ParamName string
	Routes    Routes
}

type RedirectPath struct {
	Path       string
	To         string
	Desination string
}

func (rp RedirectPath) ServeHTTP(w ResponseWriter, r *Request) {
	to := rp.To
	isUrl := strings.Contains(to, "://")
	isAbs := strings.HasPrefix(to, "/")

	if !isUrl && !isAbs {
		to = path.Clean(path.Join(r.URL.Path, to))
	}

	http.Redirect(w, r.Request, to, http.StatusPermanentRedirect)
}

type Path struct {
	Path        string
	Method      string
	Destination string
	Handler
}

func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if len(path) == 0 || []byte(path)[0] != '/' {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	path = path[1:]

	segments := strings.Split(path, "/")

	handler, params := Routes(router).findRoute(r.Method, segments)
	if handler == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	req := &Request{
		Request: r,
		Params:  params,
	}
	rw := ResponseWriter{
		ResponseWriter: w,
	}

	handler.ServeHTTP(rw, req)
}

func (routes Routes) findRoute(method string, segments []string) (Handler, url.Values) {
	var matchAllHandler Handler
	var matchAllParams url.Values

	extractFormatFromPath := func(segment string) (path, extension string) {
		s := strings.SplitN(segment, ".", 2)
		if len(s) == 0 {
			return "", ""
		}
		if len(s) == 1 {
			return s[0], ""
		}
		return s[0], s[1]
	}

	for _, r := range routes {
		switch r := r.(type) {
		case Path:
			// Remove extension
			routeMethod := r.Method
			if routeMethod == "" {
				routeMethod = "GET"
			}
			path, format := extractFormatFromPath(segments[0])

			if len(segments) != 1 || path != r.Path || routeMethod != method {
				continue
			}
			params := make(url.Values)
			if format != "" {
				params.Set("format", format)
			}
			return r.Handler, params
		case CatchAllPath:
			routeMethod := r.Method
			if routeMethod == "" {
				routeMethod = "GET"
			}

			id, format := extractFormatFromPath(segments[0])

			if len(segments) != 1 || routeMethod != method {
				continue
			}
			params := make(url.Values)
			if format != "" {
				params.Set("format", format)
			}
			params.Set(r.ParamName, id)
			matchAllHandler = r.Handler
			matchAllParams = params
		case Prefix:
			if len(segments) < 2 {
				continue
			}
			h, p := r.Routes.findRoute(method, segments[1:])
			if h != nil {
				return h, p
			}
		case CatchAllPrefix:
			if len(segments) < 2 {
				continue
			}
			matchAllHandler, matchAllParams = r.Routes.findRoute(method, segments[1:])
			if matchAllHandler == nil {
				continue
			}
			if matchAllParams == nil {
				matchAllParams = make(url.Values)
			}
			matchAllParams.Set(r.ParamName, segments[0])
		case RedirectPath:
			if len(segments) != 1 || segments[0] != r.Path {
				continue
			}
			return r, nil
		case Routes:
			h, p := r.findRoute(method, segments)
			if h != nil {
				return h, p
			}
		}
	}

	return matchAllHandler, matchAllParams
}
