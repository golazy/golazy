package pty

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"golazy.dev/lazytui/encoding/tty"
)

var (
	ErrInvalidCommand = errors.New("invalid pty command")
	ErrNotStarted     = errors.New("pty command not started")
	ErrAlreadyStarted = errors.New("pty command already started")
)

// Cmd is a command connected to a pseudo terminal.
type Cmd struct {
	Path string
	Args []string
	Env  []string
	Dir  string
	Size tty.Size

	Stdin  io.Reader
	Stdout io.Writer

	ctx context.Context

	mu     sync.Mutex
	cmd    *exec.Cmd
	master *os.File
	done   chan struct{}
}

// Command returns a new pseudo-terminal command.
func Command(path string, args ...string) *Cmd {
	return CommandContext(context.Background(), path, args...)
}

// CommandContext returns a new pseudo-terminal command with a context.
func CommandContext(ctx context.Context, path string, args ...string) *Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Cmd{
		Path: path,
		Args: append([]string(nil), args...),
		Size: tty.Size{Rows: 24, Cols: 80},
		ctx:  ctx,
	}
}

// Master returns the master side of the pseudo terminal after Start.
func (c *Cmd) Master() *os.File {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.master
}

// Process returns the started process, if any.
func (c *Cmd) Process() *os.Process {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd == nil {
		return nil
	}
	return c.cmd.Process
}

// Start starts the command connected to a pseudo terminal.
func (c *Cmd) Start() error {
	if c == nil || c.Path == "" {
		return ErrInvalidCommand
	}
	if !c.Size.Valid() {
		return tty.ErrInvalidSize
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil {
		return ErrAlreadyStarted
	}

	master, slave, err := open(c.Size)
	if err != nil {
		return err
	}
	defer slave.Close()

	cmd := exec.CommandContext(c.ctx, c.Path, c.Args...)
	cmd.Env = c.Env
	cmd.Dir = c.Dir
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	setControllingTerminal(cmd, slave)

	if err := cmd.Start(); err != nil {
		master.Close()
		return fmt.Errorf("%s: %w", c.Path, err)
	}

	c.cmd = cmd
	c.master = master
	c.done = make(chan struct{})
	c.startCopyLocked()
	return nil
}

// Run starts the command and waits for it to finish.
func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

// Wait waits for the command to finish and closes the pseudo-terminal master.
func (c *Cmd) Wait() error {
	c.mu.Lock()
	cmd := c.cmd
	master := c.master
	done := c.done
	c.mu.Unlock()

	if cmd == nil {
		return ErrNotStarted
	}

	err := cmd.Wait()
	if master != nil {
		_ = master.Close()
	}
	if done != nil {
		<-done
	}
	return err
}

// Resize changes the child terminal size.
func (c *Cmd) Resize(size tty.Size) error {
	if !size.Valid() {
		return tty.ErrInvalidSize
	}

	c.mu.Lock()
	master := c.master
	c.mu.Unlock()
	if master == nil {
		return ErrNotStarted
	}
	return resize(master, size)
}

func (c *Cmd) startCopyLocked() {
	if c.Stdin == nil && c.Stdout == nil {
		close(c.done)
		return
	}

	var wg sync.WaitGroup
	if c.Stdin != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(c.master, c.Stdin)
		}()
	}
	if c.Stdout != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = io.Copy(c.Stdout, c.master)
		}()
	}
	go func() {
		wg.Wait()
		close(c.done)
	}()
}
