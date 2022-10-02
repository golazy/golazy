package lazyaction

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

type Controller struct {
}

type ResponseWriter struct {
	http.ResponseWriter
}

type Request struct {
	*http.Request
}

type controller interface{}

type path string
type route string
type verb string
type action reflect.Method

type memberFunc func(string, ResponseWriter, *Request)
type collectionFunc func(ResponseWriter, *Request)

type resourceRoutes struct {
	path        string
	c           controller
	name        string
	collection  map[path]map[verb]action
	member      map[path]map[verb]action
	subResource []resourceRoutes
}

func (rr *resourceRoutes) call(method action, args ...interface{}) {
	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	reflect.ValueOf(rr.c).MethodByName(method.Name).Call(inputs)
}

func (rr *resourceRoutes) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if !strings.HasPrefix(p, "/"+rr.path) {
		http.NotFound(w, r)
		return
	}
	p = p[len(rr.path)+1:] // Remove trailing resource name prefix

	if strings.HasSuffix(p, "/") {
		http.Redirect(w, r, p[0:len(p)-1], http.StatusFound)
		return
	}

	slashes := strings.Count(p, "/")
	if slashes < 2 {
		if verbs, ok := rr.collection[path(p)]; ok {
			for v, a := range verbs {
				if r.Method == string(v) {
					rr.call(a, w, r)
					return
				}
			}
		}
	}

	if slashes == 1 || slashes == 2 {
		var id string
		pos := strings.Index(p[1:], "/")
		if pos == -1 {
			id = p[1:]
			p = ""
		} else {
			id = p[1 : pos+1]
			p = p[pos+1:]
		}

		// Let's try with an id

		if verbs, ok := rr.member[path(p)]; ok {
			for v, a := range verbs {
				if r.Method == string(v) {
					rr.call(a, id, w, r)
					return
				}
			}
		}
	}
	http.NotFound(w, r)
}

func (rr *resourceRoutes) addCollection(p path, v verb, a action) {
	if _, ok := rr.collection[p]; !ok {
		rr.collection[p] = make(map[verb]action)
	}
	rr.collection[p][v] = a
}

func (rr *resourceRoutes) addMember(p path, v verb, a action) {
	if _, ok := rr.member[p]; !ok {
		rr.member[p] = make(map[verb]action)
	}
	rr.member[p][v] = a
}

func (rr *resourceRoutes) String() string {
	routes := []string{""}

	appendPath := func(p path, v verb, a action) {
		routes = append(routes, fmt.Sprintf("%6s /%s%-20s %s#%s", v, rr.path, p, rr.name, a.Name))
	}

	for p, verbs := range rr.collection {
		for v, a := range verbs {
			appendPath(p, v, a)
		}
	}

	for p, verbs := range rr.member {
		for v, a := range verbs {
			appendPath("/:id"+p, v, a)
		}
	}
	return strings.Join(routes, "\n")
}

func buildResourceRoutes(controller interface{}) *resourceRoutes {

	cType := reflect.TypeOf(controller)
	fullName := cType.Elem().Name()
	name := toSnakeCase(fullName)
	if strings.HasSuffix(name, "_controller") {
		name = name[0 : len(name)-len("_controller")]
	}

	rr := &resourceRoutes{
		name:       fullName,
		path:       name,
		c:          controller,
		collection: make(map[path]map[verb]action),
		member:     make(map[path]map[verb]action),
	}

	for i := 0; i < cType.NumMethod(); i++ {
		m := cType.Method(i)
		inputs := m.Type.NumIn()
		// Todo check the method arguments comparing with memberFunc and collectionFunc
		// Show a warning and ignore in case the arguments missmatch
		if inputs == 3 {
			switch {
			case m.Name == "Index":
				rr.addCollection("", "GET", action(m))
			case m.Name == "New":
				rr.addCollection("/new", "GET", action(m))
			case m.Name == "Create":
				rr.addCollection("", "POST", action(m))
			default:
				p, v := getPathAndVerb(m.Name)
				rr.addCollection(path(p), verb(v), action(m))
			}
			continue
		}
		if inputs == 4 {
			switch {
			case m.Name == "Show":
				rr.addMember("", "GET", action(m))
			case m.Name == "Update":
				rr.addMember("", "PUT", action(m))
				rr.addMember("", "PATCH", action(m))
			case m.Name == "Delete":
				rr.addMember("", "DELETE", action(m))
			default:
				p, v := getPathAndVerb(m.Name)
				rr.addMember(path(p), verb(v), action(m))
			}
			continue
		}
		/*
			mfType := reflect.TypeOf((memberFunc)(nil))
			cfType := reflect.TypeOf((collectionFunc)(nil))

					if m.Type.Implements(mfType) {
						panic("yeah")
						continue
					}
					if m.Type.Implements(cfType) {
						panic("yeah")
						continue
					}
					panic("oh no")
		*/
	}
	return rr
}

var prefixes = []string{"Get", "Post", "Delete", "Patch", "Put"}

func getPathAndVerb(name string) (path, verb string) {
	path = name
	verb = "GET"
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			verb = strings.ToUpper(prefix)
			path = path[len(prefix):]
		}
	}
	path = "/" + toSnakeCase(path)
	return
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
