package lazydeps

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"
)

const appNode = "app"

type Func[T any] func(context.Context) (context.Context, T, error, context.CancelFunc)

type Scope struct {
	mu      sync.Mutex
	ctx     context.Context
	nodes   map[string]*node
	edges   map[string]map[string]struct{}
	current []string
	nextID  int
	logger  *slog.Logger
}

type node struct {
	name          string
	id            int
	cancelContext func(error)
	stop          context.CancelFunc
	once          sync.Once
	done          chan struct{}
}

type Ref[T any] struct {
	scope *Scope
	name  string
	value T
}

type Graph struct {
	Nodes []string `json:"nodes"`
	Edges []Edge   `json:"edges"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Option func(*Scope)

func WithLogger(logger *slog.Logger) Option {
	return func(u *Scope) {
		if logger != nil {
			u.logger = logger
		}
	}
}

func New(ctx context.Context, opts ...Option) *Scope {
	if ctx == nil {
		ctx = context.Background()
	}
	u := &Scope{
		ctx:   ctx,
		nodes: map[string]*node{appNode: {name: appNode, done: closedNodeDone()}},
		edges: make(map[string]map[string]struct{}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(u)
		}
	}
	if u.logger == nil {
		u.logger = slog.Default()
	}
	return u
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

	serviceCtx, cancelCause := context.WithCancelCause(u.Context())
	serviceCtx = withCurrent(serviceCtx, name)

	u.begin(name)
	defer u.end(name)
	nextCtx, value, err, stop := fn(serviceCtx)
	if err != nil {
		cancelCause(err)
		if stop != nil {
			stop()
		}
		return zero, err
	}

	u.addNode(name, cancelCause, stop)
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

func (u *Scope) Shutdown(ctx context.Context, reason string) error {
	if u == nil {
		return fmt.Errorf("lazydeps: nil Scope")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if reason == "" {
		reason = "shutdown"
	}

	var errs []error
	for _, node := range u.shutdownOrder() {
		select {
		case <-ctx.Done():
			return errors.Join(append(errs, ctx.Err())...)
		default:
		}

		logger := u.loggerForUse()
		logger.Info("lazydeps: canceling service context", "service", node.name, "reason", reason)
		started := time.Now()
		finished := make(chan struct{})
		go func() {
			node.shutdown(errors.New(reason))
			close(finished)
		}()

		select {
		case <-finished:
			logger.Info("lazydeps: service cleanup finished", "service", node.name, "duration", time.Since(started).String())
		case <-ctx.Done():
			elapsed := time.Since(started)
			err := fmt.Errorf("lazydeps: service %q cleanup interrupted after %s: %w", node.name, elapsed, ctx.Err())
			logger.Error("lazydeps: service cleanup interrupted", "service", node.name, "duration", elapsed.String(), "err", ctx.Err())
			errs = append(errs, err)
			return errors.Join(errs...)
		}
	}
	return errors.Join(errs...)
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

func (u *Scope) addNode(name string, cancelContext func(error), stop context.CancelFunc) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.nextID++
	u.nodes[name] = &node{
		name:          name,
		id:            u.nextID,
		cancelContext: cancelContext,
		stop:          stop,
		done:          make(chan struct{}),
	}
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

func (u *Scope) shutdownOrder() []*node {
	u.mu.Lock()
	defer u.mu.Unlock()

	indegree := make(map[string]int, len(u.nodes))
	outgoing := make(map[string][]string, len(u.edges))
	for name := range u.nodes {
		if name != appNode {
			indegree[name] = 0
		}
	}
	for from, targets := range u.edges {
		if from == appNode {
			continue
		}
		if _, ok := indegree[from]; !ok {
			continue
		}
		for to := range targets {
			if _, ok := indegree[to]; !ok {
				continue
			}
			outgoing[from] = append(outgoing[from], to)
			indegree[to]++
		}
	}
	for from := range outgoing {
		slices.SortFunc(outgoing[from], func(a, b string) int {
			return compareNodeShutdownOrder(u.nodes[a], u.nodes[b])
		})
	}

	ready := make([]string, 0, len(indegree))
	for name, count := range indegree {
		if count == 0 {
			ready = append(ready, name)
		}
	}
	sortNodeNamesForShutdown(u.nodes, ready)

	order := make([]*node, 0, len(indegree))
	for len(ready) > 0 {
		name := ready[0]
		ready = ready[1:]
		order = append(order, u.nodes[name])
		for _, target := range outgoing[name] {
			indegree[target]--
			if indegree[target] == 0 {
				ready = append(ready, target)
				sortNodeNamesForShutdown(u.nodes, ready)
			}
		}
		delete(indegree, name)
	}

	if len(indegree) > 0 {
		remaining := make([]string, 0, len(indegree))
		for name := range indegree {
			remaining = append(remaining, name)
		}
		sortNodeNamesForShutdown(u.nodes, remaining)
		for _, name := range remaining {
			order = append(order, u.nodes[name])
		}
	}
	return order
}

func sortNodeNamesForShutdown(nodes map[string]*node, names []string) {
	slices.SortFunc(names, func(a, b string) int {
		return compareNodeShutdownOrder(nodes[a], nodes[b])
	})
}

func compareNodeShutdownOrder(a, b *node) int {
	if a.id > b.id {
		return -1
	}
	if a.id < b.id {
		return 1
	}
	if a.name < b.name {
		return -1
	}
	if a.name > b.name {
		return 1
	}
	return 0
}

func (u *Scope) loggerForUse() *slog.Logger {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.logger == nil {
		return slog.Default()
	}
	return u.logger
}

func (n *node) shutdown(cause error) {
	if n == nil {
		return
	}
	n.once.Do(func() {
		if n.cancelContext != nil {
			n.cancelContext(cause)
		}
		if n.stop != nil {
			n.stop()
		}
		close(n.done)
	})
	<-n.done
}

func closedNodeDone() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}
