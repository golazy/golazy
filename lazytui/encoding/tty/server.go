package tty

import (
	"context"
	"fmt"
	"sync"
)

// Server owns terminal state and applies requests from clients.
type Server struct {
	device *Device

	mu     sync.Mutex
	states map[StateID]*State
	next   uint64
}

func NewServer(device *Device) (*Server, error) {
	if device == nil || device.backend == nil {
		return nil, ErrNilBackend
	}
	return &Server{
		device: device,
		states: make(map[StateID]*State),
	}, nil
}

// Serve applies a single request.
func (s *Server) Serve(ctx context.Context, req Request) Response {
	if err := ctx.Err(); err != nil {
		return errorResponse(err)
	}
	if s == nil || s.device == nil {
		return errorResponse(ErrNilBackend)
	}

	switch req.Op {
	case OpIsTerminal:
		return Response{IsTerminal: s.device.IsTerminal()}
	case OpSize:
		size, err := s.device.Size()
		if err != nil {
			return errorResponse(err)
		}
		return Response{Size: size}
	case OpResize:
		if err := s.device.Resize(req.Size); err != nil {
			return errorResponse(err)
		}
		return Response{}
	case OpMakeRaw:
		state, err := s.device.MakeRaw()
		if err != nil {
			return errorResponse(err)
		}
		id := s.storeState(state)
		return Response{State: id}
	case OpRestore:
		state, ok := s.takeState(req.State)
		if !ok {
			return errorResponse(ErrUnknownState)
		}
		if err := s.device.Restore(state); err != nil {
			return errorResponse(err)
		}
		return Response{}
	default:
		return errorResponse(fmt.Errorf("unknown tty operation %q", req.Op))
	}
}

// RoundTrip lets Server satisfy Transport for in-process clients and tests.
func (s *Server) RoundTrip(ctx context.Context, req Request) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	return s.Serve(ctx, req), nil
}

func (s *Server) storeState(state *State) StateID {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next++
	id := StateID(fmt.Sprintf("state-%d", s.next))
	s.states[id] = state
	return id
}

func (s *Server) takeState(id StateID) (*State, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.states[id]
	if !ok {
		return nil, false
	}
	delete(s.states, id)
	return state, true
}
