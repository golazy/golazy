//go:build !linux

package tty

import (
	"fmt"
	"os"
)

func openOS(file *os.File) (Backend, error) {
	if file == nil {
		return nil, fmt.Errorf("%w: nil file", ErrNilBackend)
	}
	return nil, ErrUnsupported
}
