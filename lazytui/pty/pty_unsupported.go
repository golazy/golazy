//go:build !linux

package pty

import (
	"os"
	"os/exec"

	"golazy.dev/lazytui/encoding/tty"
)

func open(size tty.Size) (*os.File, *os.File, error) {
	return nil, nil, tty.ErrUnsupported
}

func setControllingTerminal(cmd *exec.Cmd, slave *os.File) {}

func resize(file *os.File, size tty.Size) error {
	return tty.ErrUnsupported
}
