package lazyapp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

var errRuntimeDraining = errors.New("application is draining")

type runtimeState struct {
	draining          atomic.Bool
	activeRequests    atomic.Int64
	activeConnections atomic.Int64
	connections       sync.Map
}

func newRuntimeState() *runtimeState {
	return &runtimeState{}
}

func (s *runtimeState) Handler(next http.Handler) http.Handler {
	if s == nil || next == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.activeRequests.Add(1)
		defer s.activeRequests.Add(-1)
		next.ServeHTTP(w, r)
	})
}

func (s *runtimeState) ConnState(conn net.Conn, state http.ConnState) {
	if s == nil || conn == nil {
		return
	}
	switch state {
	case http.StateNew:
		if _, loaded := s.connections.LoadOrStore(conn, struct{}{}); !loaded {
			s.activeConnections.Add(1)
		}
	case http.StateHijacked, http.StateClosed:
		if _, loaded := s.connections.LoadAndDelete(conn); loaded {
			s.activeConnections.Add(-1)
		}
	}
}

func (s *runtimeState) SetDraining(draining bool) {
	if s == nil {
		return
	}
	s.draining.Store(draining)
}

func (s *runtimeState) Draining() bool {
	return s != nil && s.draining.Load()
}

func (s *runtimeState) ActiveRequests() int64 {
	if s == nil {
		return 0
	}
	return s.activeRequests.Load()
}

func (s *runtimeState) ActiveConnections() int64 {
	if s == nil {
		return 0
	}
	return s.activeConnections.Load()
}

func (s *runtimeState) ReadinessCheck(context.Context) error {
	if s.Draining() {
		return errRuntimeDraining
	}
	return nil
}
