//go:build linux

package tty

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type osBackend struct {
	fd int
}

func openOS(file *os.File) (Backend, error) {
	if file == nil {
		return nil, fmt.Errorf("%w: nil file", ErrNilBackend)
	}
	return &osBackend{fd: int(file.Fd())}, nil
}

func (b *osBackend) IsTerminal() bool {
	_, err := b.Size()
	return err == nil
}

func (b *osBackend) Size() (Size, error) {
	var ws winsize
	if err := ioctl(b.fd, syscall.TIOCGWINSZ, unsafe.Pointer(&ws)); err != nil {
		return Size{}, err
	}
	return Size{Rows: int(ws.Row), Cols: int(ws.Col)}, nil
}

func (b *osBackend) Resize(size Size) error {
	if !size.Valid() {
		return ErrInvalidSize
	}
	ws := winsize{Row: uint16(size.Rows), Col: uint16(size.Cols)}
	return ioctl(b.fd, syscall.TIOCSWINSZ, unsafe.Pointer(&ws))
}

func (b *osBackend) MakeRaw() (*State, error) {
	var old syscall.Termios
	if err := ioctl(b.fd, syscall.TCGETS, unsafe.Pointer(&old)); err != nil {
		return nil, err
	}

	raw := old
	raw.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	raw.Oflag &^= syscall.OPOST
	raw.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	raw.Cflag &^= syscall.CSIZE | syscall.PARENB
	raw.Cflag |= syscall.CS8
	raw.Cc[syscall.VMIN] = 1
	raw.Cc[syscall.VTIME] = 0

	if err := ioctl(b.fd, syscall.TCSETS, unsafe.Pointer(&raw)); err != nil {
		return nil, err
	}
	return newState(&old), nil
}

func (b *osBackend) Restore(state *State) error {
	termios, ok := state.value.(*syscall.Termios)
	if !ok || termios == nil {
		return ErrUnknownState
	}
	return ioctl(b.fd, syscall.TCSETS, unsafe.Pointer(termios))
}

type winsize struct {
	Row    uint16
	Col    uint16
	XPixel uint16
	YPixel uint16
}

func ioctl(fd int, req uint, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}
