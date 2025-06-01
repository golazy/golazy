package router

import (
	"net/http"
	"strings"
)

// PathMatcher holds information for routing urls with wildcard
type PathMatcher[T any] struct {
	root pathNode[T]
}

func NewPathMatcher[T any]() Matcher[T] {
	return &PathMatcher[T]{
		root: pathNode[T]{},
	}
}

// Add a new route to the table
func (rt *PathMatcher[T]) Add(req *RouteDefinition, t *T) {
	path := req.Path
	if len(path) == 0 {
		path = "/"
	}
	if path[0] != '/' {
		panic("err: while trying to add route " + path + " to route table. Path should start with /")
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		panic("Path can't end with /. Got " + path)
	}
	if strings.Contains(path, "*") && !strings.HasSuffix(path, "/*") {
		panic("Path can only contain * at the end. Got " + path)
	}
	node := &rt.root
	for _, s := range strings.Split(path[1:], "/") {
		if s == "*" {
			node.isWildcard = true
			break
		}
		node = node.findOrCreate(s)
	}
	if node.leaf != nil {

		panic("Route for " + path + " was already defined")
	}
	node.leaf = t
}

func (n *PathMatcher[T]) Find(req *http.Request) *T {
	// TODO Filter by extension
	path := req.URL.EscapedPath()
	if len(path) == 0 {
		path = "/"
	}
	if path[0] != '/' {
		panic("path should start with /")
	}
	found := n.root.find(path[1:])
	return found
}

type pathNode[T any] struct {
	name         string
	staticNodes  []*pathNode[T]
	dynamicNodes []*pathNode[T]
	leaf         *T
	isWildcard   bool
}

func (n *pathNode[T]) findOrCreate(name string) *pathNode[T] {
	if strings.HasPrefix(name, ":") {
		for _, child := range n.dynamicNodes {
			if child.name == name {
				return child
			}
		}
		child := &pathNode[T]{
			name:         name,
			staticNodes:  []*pathNode[T]{},
			dynamicNodes: []*pathNode[T]{},
		}

		n.dynamicNodes = append(n.dynamicNodes, child)
		return child
	}
	for _, child := range n.staticNodes {
		if child.name == name {
			return child
		}
	}
	child := &pathNode[T]{
		name:         name,
		staticNodes:  []*pathNode[T]{},
		dynamicNodes: []*pathNode[T]{},
	}

	n.staticNodes = append(n.staticNodes, child)
	return child

}

type routeInfo[T any] struct {
	path string
	t    *T
}

func (n pathNode[T]) Routes() []routeInfo[T] {
	var rl []routeInfo[T]
	for _, child := range n.staticNodes {
		for _, s := range child.Routes() {
			rl = append(rl, routeInfo[T]{path: n.name + "/" + s.path, t: s.t})
		}
	}
	for _, child := range n.dynamicNodes {
		for _, s := range child.Routes() {
			rl = append(rl, routeInfo[T]{path: n.name + "/" + s.path, t: s.t})
		}
	}
	if n.leaf != nil {
		rl = append(rl, routeInfo[T]{path: n.name, t: n.leaf})
	}
	return rl

}

func (rt *PathMatcher[T]) Routes() []routeInfo[T] {
	return rt.root.Routes()
}

func (rt *PathMatcher[T]) All() []Route[T] {
	var all []Route[T]
	for _, r := range rt.Routes() {
		req, err := http.NewRequest("GET", r.path, nil)
		if err != nil {
			panic(err)
		}
		all = append(all, Route[T]{
			Req: req,
			T:   r.t,
		})
	}
	return all
}

func (n *pathNode[T]) find(path string) *T {
	if n.isWildcard {
		return n.leaf
	}
	name := path
	rest := ""
	i := strings.IndexRune(path, '/')
	if i != -1 {
		name = path[:i]
		rest = path[i+1:]
	}
	for _, child := range n.staticNodes {
		if child.name == name {
			if len(rest) == 0 && child.leaf != nil {
				return child.leaf
			}
			if len(rest) > 0 {
				t := child.find(rest)
				if t != nil {
					return t
				}
			}
		}
	}
	for _, child := range n.dynamicNodes {
		if len(rest) == 0 && child.leaf != nil {
			return child.leaf
		}
		if len(rest) > 0 {
			t := child.find(rest)
			if t != nil {
				return t
			}
		}
	}
	return nil
}
