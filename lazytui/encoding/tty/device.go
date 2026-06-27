package tty

import "os"

// Backend applies terminal operations to an implementation such as a real TTY,
// a ConPTY object, or a test fake.
type Backend interface {
	IsTerminal() bool
	Size() (Size, error)
	Resize(Size) error
	MakeRaw() (*State, error)
	Restore(*State) error
}

// Device is the local handle for terminal-control operations.
type Device struct {
	backend Backend
}

// NewDevice wraps backend in a Device.
func NewDevice(backend Backend) (*Device, error) {
	if backend == nil {
		return nil, ErrNilBackend
	}
	return &Device{backend: backend}, nil
}

// Open wraps a real OS terminal file.
func Open(file *os.File) (*Device, error) {
	backend, err := openOS(file)
	if err != nil {
		return nil, err
	}
	return NewDevice(backend)
}

// IsTerminal reports whether the device is backed by a terminal.
func (d *Device) IsTerminal() bool {
	if d == nil || d.backend == nil {
		return false
	}
	return d.backend.IsTerminal()
}

// Size returns the current terminal size.
func (d *Device) Size() (Size, error) {
	if d == nil || d.backend == nil {
		return Size{}, ErrNilBackend
	}
	return d.backend.Size()
}

// Resize sets the terminal size.
func (d *Device) Resize(size Size) error {
	if d == nil || d.backend == nil {
		return ErrNilBackend
	}
	if !size.Valid() {
		return ErrInvalidSize
	}
	return d.backend.Resize(size)
}

// MakeRaw puts the terminal into raw mode and returns a restorable state.
func (d *Device) MakeRaw() (*State, error) {
	if d == nil || d.backend == nil {
		return nil, ErrNilBackend
	}
	return d.backend.MakeRaw()
}

// Restore restores a previous terminal state.
func (d *Device) Restore(state *State) error {
	if d == nil || d.backend == nil {
		return ErrNilBackend
	}
	return d.backend.Restore(state)
}
