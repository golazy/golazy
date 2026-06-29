//go:build lazydev

package lazydeps

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazysse"
)

const LazyDevDependenciesPath = "/dependencies"
const LazyDevDependencyShutdownPath = "/dependencies/shutdown"
const LazyDevDependencyShutdownEventsPath = "/dependencies/shutdown/events"

const lazyDevShutdownDelay = time.Second

// LazyDevRuntime reports application runtime state to dependency development
// handlers.
type LazyDevRuntime interface {
	SetDraining(bool)
	Draining() bool
	ActiveRequests() int64
	ActiveConnections() int64
}

type LazyDevOption func(*lazyDevOptions)

type lazyDevOptions struct {
	runtime LazyDevRuntime
}

// WithLazyDevRuntime lets lazydev dependency handlers report readiness,
// active requests, and active connections while simulating shutdown.
func WithLazyDevRuntime(runtime LazyDevRuntime) LazyDevOption {
	return func(options *lazyDevOptions) {
		options.runtime = runtime
	}
}

type LazyDevShutdownState struct {
	Graph             Graph                 `json:"graph"`
	Ready             bool                  `json:"ready"`
	ReadyStatus       int                   `json:"ready_status"`
	ReadyText         string                `json:"ready_text"`
	ActiveRequests    int64                 `json:"active_requests"`
	ActiveConnections int64                 `json:"active_connections"`
	Running           bool                  `json:"running"`
	Phase             string                `json:"phase"`
	Message           string                `json:"message"`
	Nodes             []LazyDevShutdownNode `json:"nodes"`
	Error             string                `json:"error,omitempty"`
}

type LazyDevShutdownNode struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// RegisterLazyDevHandlers registers dependency graph inspection endpoints.
func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, dependencies *Scope, opts ...LazyDevOption) {
	if controlPlane == nil {
		return
	}
	var options lazyDevOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	shutdowns := newLazyDevShutdownSimulator(dependencies, options.runtime)
	controlPlane.Handle("GET "+LazyDevDependenciesPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		graph := Graph{Nodes: []string{}, Edges: []Edge{}}
		if dependencies != nil {
			graph = dependencies.Graph()
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(graph); err != nil {
			http.Error(w, fmt.Sprintf("dependencies: %v", err), http.StatusInternalServerError)
		}
	}))
	controlPlane.Handle("GET "+LazyDevDependencyShutdownPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeLazyDevJSON(w, shutdowns.State())
	}))
	controlPlane.Handle("POST "+LazyDevDependencyShutdownPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delay := parseLazyDevShutdownDelay(r)
		state := shutdowns.Start(delay)
		writeLazyDevJSON(w, state)
	}))
	controlPlane.Handle("GET "+LazyDevDependencyShutdownEventsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream, err := lazysse.Start(w, r)
		if err != nil {
			http.Error(w, fmt.Sprintf("dependencies shutdown stream: %v", err), http.StatusInternalServerError)
			return
		}
		defer stream.Close()
		stream.Heartbeat(15 * time.Second)
		if err := stream.JSON("shutdown", shutdowns.State()); err != nil {
			return
		}
		events, unsubscribe := shutdowns.Subscribe()
		defer unsubscribe()
		for {
			select {
			case <-stream.Done():
				return
			case state, ok := <-events:
				if !ok {
					return
				}
				if err := stream.JSON("shutdown", state); err != nil {
					return
				}
			}
		}
	}))
}

func writeLazyDevJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, fmt.Sprintf("dependencies: %v", err), http.StatusInternalServerError)
	}
}

func parseLazyDevShutdownDelay(r *http.Request) time.Duration {
	if r == nil {
		return 0
	}
	value := r.URL.Query().Get("delay_seconds")
	if value == "" {
		value = r.FormValue("delay_seconds")
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return 0
	}
	if seconds > 120 {
		seconds = 120
	}
	return time.Duration(seconds) * time.Second
}

type lazyDevShutdownSimulator struct {
	dependencies *Scope
	runtime      LazyDevRuntime
	mu           sync.Mutex
	running      bool
	phase        string
	message      string
	err          string
	nodeStates   map[string]string
	subscribers  map[chan LazyDevShutdownState]struct{}
}

func newLazyDevShutdownSimulator(dependencies *Scope, runtime LazyDevRuntime) *lazyDevShutdownSimulator {
	return &lazyDevShutdownSimulator{
		dependencies: dependencies,
		runtime:      runtime,
		phase:        "idle",
		message:      "Shutdown simulation has not started.",
		nodeStates:   map[string]string{},
		subscribers:  map[chan LazyDevShutdownState]struct{}{},
	}
}

func (s *lazyDevShutdownSimulator) State() LazyDevShutdownState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateLocked()
}

func (s *lazyDevShutdownSimulator) Subscribe() (<-chan LazyDevShutdownState, func()) {
	ch := make(chan LazyDevShutdownState, 16)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subscribers, ch)
		close(ch)
		s.mu.Unlock()
	}
}

func (s *lazyDevShutdownSimulator) Start(delay time.Duration) LazyDevShutdownState {
	s.mu.Lock()
	if s.running {
		state := s.stateLocked()
		s.mu.Unlock()
		return state
	}
	s.running = true
	s.err = ""
	s.nodeStates = map[string]string{}
	if delay > 0 {
		s.phase = "traffic"
		s.message = fmt.Sprintf("Sending test traffic for %s before shutdown.", delay.Round(time.Second))
	} else {
		s.phase = "draining"
		s.message = "Starting shutdown simulation."
	}
	state := s.broadcastLocked()
	s.mu.Unlock()

	go s.run(delay)
	return state
}

func (s *lazyDevShutdownSimulator) run(delay time.Duration) {
	if delay > 0 {
		deadline := time.NewTimer(delay)
		ticker := time.NewTicker(250 * time.Millisecond)
		for waiting := true; waiting; {
			select {
			case <-deadline.C:
				waiting = false
			case <-ticker.C:
				s.mu.Lock()
				s.broadcastLocked()
				s.mu.Unlock()
			}
		}
		ticker.Stop()
	}

	if s.runtime != nil {
		s.runtime.SetDraining(true)
	}
	s.mu.Lock()
	s.phase = "draining"
	s.message = "GET /readyz is now 503; waiting for active requests to finish."
	s.nodeStates["app"] = "draining"
	s.broadcastLocked()
	s.mu.Unlock()

	ticker := time.NewTicker(250 * time.Millisecond)
	for {
		if s.activeRequests() == 0 {
			break
		}
		<-ticker.C
		s.mu.Lock()
		s.broadcastLocked()
		s.mu.Unlock()
	}
	ticker.Stop()

	s.mu.Lock()
	s.phase = "shutdown"
	s.message = "No active requests remain; canceling dependencies."
	s.broadcastLocked()
	s.mu.Unlock()

	var err error
	if s.dependencies != nil {
		err = s.dependencies.shutdown(
			context.Background(),
			"lazydev shutdown simulation",
			shutdownOptions{
				Delay: lazyDevShutdownDelay,
				Observer: func(event shutdownEvent) {
					s.mu.Lock()
					s.nodeStates[event.Service] = event.State
					switch event.State {
					case "canceling":
						s.phase = "shutdown"
						s.message = "Canceling " + event.Service + "."
					case "stopped":
						s.message = event.Service + " stopped."
					case "interrupted":
						s.phase = "error"
						s.message = event.Service + " shutdown interrupted."
					}
					s.broadcastLocked()
					s.mu.Unlock()
				},
			},
		)
	}

	s.mu.Lock()
	s.running = false
	if err != nil {
		s.phase = "error"
		s.err = err.Error()
		s.message = "Shutdown simulation failed."
	} else {
		s.phase = "complete"
		s.message = "Shutdown simulation complete."
	}
	s.broadcastLocked()
	s.mu.Unlock()
}

func (s *lazyDevShutdownSimulator) stateLocked() LazyDevShutdownState {
	graph := Graph{Nodes: []string{}, Edges: []Edge{}}
	if s.dependencies != nil {
		graph = s.dependencies.Graph()
	}
	ready := true
	if s.runtime != nil {
		ready = !s.runtime.Draining()
	}
	status := http.StatusOK
	readyText := "GET /readyz => 200 ready"
	if !ready {
		status = http.StatusServiceUnavailable
		readyText = "GET /readyz => 503 not ready"
	}

	nodes := make([]LazyDevShutdownNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		name := string(node)
		state := s.nodeStates[name]
		if state == "" {
			state = "running"
		}
		if name == "app" && !ready && state == "running" {
			state = "draining"
		}
		nodes = append(nodes, LazyDevShutdownNode{Name: name, State: state})
	}
	return LazyDevShutdownState{
		Graph:             graph,
		Ready:             ready,
		ReadyStatus:       status,
		ReadyText:         readyText,
		ActiveRequests:    s.activeRequests(),
		ActiveConnections: s.activeConnections(),
		Running:           s.running,
		Phase:             s.phase,
		Message:           s.message,
		Nodes:             nodes,
		Error:             s.err,
	}
}

func (s *lazyDevShutdownSimulator) broadcastLocked() LazyDevShutdownState {
	state := s.stateLocked()
	for ch := range s.subscribers {
		select {
		case ch <- state:
		default:
		}
	}
	return state
}

func (s *lazyDevShutdownSimulator) activeRequests() int64 {
	if s.runtime == nil {
		return 0
	}
	return s.runtime.ActiveRequests()
}

func (s *lazyDevShutdownSimulator) activeConnections() int64 {
	if s.runtime == nil {
		return 0
	}
	return s.runtime.ActiveConnections()
}
