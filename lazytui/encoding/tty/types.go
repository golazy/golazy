package tty

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidSize  = errors.New("invalid terminal size")
	ErrNilBackend   = errors.New("nil tty backend")
	ErrNilTransport = errors.New("nil tty transport")
	ErrUnknownState = errors.New("unknown terminal state")
	ErrUnsupported  = errors.New("terminal operation unsupported")
)

// Size is a terminal size in character cells.
type Size struct {
	Rows int `json:"rows,omitempty"`
	Cols int `json:"cols,omitempty"`
}

// Valid reports whether the size can be applied to a terminal.
func (s Size) Valid() bool {
	return s.Rows > 0 && s.Cols > 0
}

func (s Size) String() string {
	return fmt.Sprintf("%dx%d", s.Rows, s.Cols)
}

// StateID is a server-local token for a saved terminal state.
type StateID string

// State contains backend-specific terminal state.
//
// State values intentionally do not encode across the wire. A Server stores
// states and returns StateID tokens to clients.
type State struct {
	value any
}

func newState(value any) *State {
	return &State{value: value}
}
