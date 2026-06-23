package lazydeps

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

const appNode = "app"

type Func[T any] func(context.Context) (context.Context, T, error, context.CancelFunc)

type Scope struct {
	mu      sync.Mutex
	ctx     context.Context
	nodes   map[string]*node
	edges   map[string]map[string]struct{}
	current []string
}

type node struct {
	name   string
	cancel context.CancelFunc
}

type Ref[T any] struct {
	scope *Scope
	name  string
	value T
}

type Graph struct {
	Nodes []string
	Edges []Edge
}

type Edge struct {
	From string
	To   string
}

func New(ctx context.Context) *Scope {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Scope{
		ctx:   ctx,
		nodes: map[string]*node{appNode: {name: appNode}},
		edges: make(map[string]map[string]struct{}),
	}
}

func (u *Scope) Context() context.Context {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.ctx
}

func (u *Scope) SetContext(ctx context.Context) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.ctx = ctx
}

func Service[T any](u *Scope, name string, fn Func[T]) (Ref[T], error) {
	var zero Ref[T]
	if u == nil {
		return zero, fmt.Errorf("lazydeps: nil Scope")
	}
	if name == "" {
		return zero, fmt.Errorf("lazydeps: service name is required")
	}
	if fn == nil {
		return zero, fmt.Errorf("lazydeps: service %q initializer is nil", name)
	}

	serviceCtx, cancelCtx := context.WithCancel(u.Context())
	serviceCtx = withCurrent(serviceCtx, name)

	u.begin(name)
	defer u.end(name)
	nextCtx, value, err, stop := fn(serviceCtx)
	if err != nil {
		cancelCtx()
		if stop != nil {
			stop()
		}
		return zero, err
	}

	cancel := func() {
		cancelCtx()
		if stop != nil {
			stop()
		}
	}
	u.addNode(name, cancel)
	u.addEdge(appNode, name)
	u.SetContext(nextCtx)
	return Ref[T]{scope: u, name: name, value: value}, nil
}

func (r Ref[T]) Use() T {
	if r.scope == nil {
		panic("lazydeps: zero Ref used")
	}
	current := r.scope.currentNode()
	if current == "" {
		panic(fmt.Sprintf("lazydeps: %s used outside service initialization", r.name))
	}
	r.scope.addEdge(current, r.name)
	return r.value
}

func (u *Scope) Graph() Graph {
	u.mu.Lock()
	defer u.mu.Unlock()

	graph := Graph{
		Nodes: make([]string, 0, len(u.nodes)),
	}
	for name := range u.nodes {
		graph.Nodes = append(graph.Nodes, name)
	}
	slices.Sort(graph.Nodes)

	for from, targets := range u.edges {
		for to := range targets {
			graph.Edges = append(graph.Edges, Edge{From: from, To: to})
		}
	}
	slices.SortFunc(graph.Edges, func(a Edge, b Edge) int {
		if a.From < b.From {
			return -1
		}
		if a.From > b.From {
			return 1
		}
		if a.To < b.To {
			return -1
		}
		if a.To > b.To {
			return 1
		}
		return 0
	})
	return graph
}

func (u *Scope) begin(name string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.current = append(u.current, name)
}

func (u *Scope) end(name string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.current) == 0 {
		return
	}
	last := len(u.current) - 1
	if u.current[last] == name {
		u.current = u.current[:last]
		return
	}
	for index := last; index >= 0; index-- {
		if u.current[index] == name {
			u.current = slices.Delete(u.current, index, index+1)
			return
		}
	}
}

func (u *Scope) currentNode() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.current) == 0 {
		return ""
	}
	return u.current[len(u.current)-1]
}

func (u *Scope) addNode(name string, cancel context.CancelFunc) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.nodes[name] = &node{name: name, cancel: sync.OnceFunc(cancel)}
}

func (u *Scope) addEdge(from string, to string) {
	if from == "" || to == "" || from == to {
		return
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.edges[from] == nil {
		u.edges[from] = make(map[string]struct{})
	}
	u.edges[from][to] = struct{}{}
}

type currentKey struct{}

func withCurrent(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, currentKey{}, name)
}
