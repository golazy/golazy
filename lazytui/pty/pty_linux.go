//go:build linux

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"unsafe"

	"golazy.dev/lazytui/encoding/tty"
)

func open(size tty.Size) (*os.File, *os.File, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			master.Close()
		}
	}()

	slaveName, err := slaveName(master)
	if err != nil {
		return nil, nil, err
	}
	if err := unlock(master); err != nil {
		return nil, nil, err
	}

	slave, err := os.OpenFile(slaveName, os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	if err := resize(slave, size); err != nil {
		slave.Close()
		return nil, nil, err
	}
	return master, slave, nil
}

func setControllingTerminal(cmd *exec.Cmd, slave *os.File) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    3,
	}
	cmd.ExtraFiles = []*os.File{slave}
}

func resize(file *os.File, size tty.Size) error {
	if !size.Valid() {
		return tty.ErrInvalidSize
	}
	ws := winsize{Row: uint16(size.Rows), Col: uint16(size.Cols)}
	return ioctl(file.Fd(), syscall.TIOCSWINSZ, unsafe.Pointer(&ws))
}

func slaveName(file *os.File) (string, error) {
	var number uint32
	if err := ioctl(file.Fd(), syscall.TIOCGPTN, unsafe.Pointer(&number)); err != nil {
		return "", fmt.Errorf("pty number: %w", err)
	}
	return "/dev/pts/" + strconv.Itoa(int(number)), nil
}

func unlock(file *os.File) error {
	var unlock int
	if err := ioctl(file.Fd(), syscall.TIOCSPTLCK, unsafe.Pointer(&unlock)); err != nil {
		return fmt.Errorf("unlock pty: %w", err)
	}
	return nil
}

type winsize struct {
	Row    uint16
	Col    uint16
	XPixel uint16
	YPixel uint16
}

func ioctl(fd uintptr, req uint, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(req), uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}
