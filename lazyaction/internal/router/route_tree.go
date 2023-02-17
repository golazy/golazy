package router

import (
	"strings"
)

type node[T any] struct {
	name         string
	staticNodes  []*node[T]
	dynamicNodes []*node[T]
	leaf         *T
}

func (n *node[T]) findOrCreate(name string) *node[T] {
	if strings.HasPrefix(name, ":") {
		for _, child := range n.dynamicNodes {
			if child.name == name {
				return child
			}
		}
		child := &node[T]{
			name:         name,
			staticNodes:  []*node[T]{},
			dynamicNodes: []*node[T]{},
		}

		n.dynamicNodes = append(n.dynamicNodes, child)
		return child
	}
	for _, child := range n.staticNodes {
		if child.name == name {
			return child
		}
	}
	child := &node[T]{
		name:         name,
		staticNodes:  []*node[T]{},
		dynamicNodes: []*node[T]{},
	}

	n.staticNodes = append(n.staticNodes, child)
	return child

}

type routeInfo[T any] struct {
	path string
	t    *T
}

func (n node[T]) Routes() []routeInfo[T] {
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

// routeTree holds information for routing urls with wildcard
type routeTree[T any] struct {
	node[T]
}

func NewRouteTable[T any]() *routeTree[T] {
	return &routeTree[T]{node: node[T]{}}
}

// Add a new route to the table
// Paths not starting with / or ending with slash results in panic
func (rt *routeTree[T]) Add(path string, dest *T) {
	if len(path) == 0 || path[0] != '/' {
		panic("Path should start with /")
	}
	if len(path) > 1 && path[len(path)-1] == '/' {
		panic("Path can't end with /. Got " + path)
	}
	node := &rt.node
	for _, s := range strings.Split(path[1:], "/") {
		node = node.findOrCreate(s)
	}
	if node.leaf != nil {
		panic("Route for " + path + " was already defined")
	}
	node.leaf = dest
}

func (rt *routeTree[T]) Routes() []routeInfo[T] {
	return rt.node.Routes()
}
func (n *routeTree[T]) Find(path string) *T {
	if len(path) < 1 || path[0] != '/' {
		panic("path should start with /")
	}
	found := n.node.find(path[1:])
	return found
}

func (n *node[T]) find(path string) *T {
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
