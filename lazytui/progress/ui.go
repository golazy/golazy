package progress

import (
	"errors"
	"io"
)

// UI gives a task explicit access to the progress-controlled terminal.
type UI struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	rawStdout   io.Writer
	rawStderr   io.Writer
	renderer    *renderer
	taskName    string
	parallelRun bool
}

// Stdin returns the task input stream.
func (u *UI) Stdin() io.Reader {
	return u.stdin
}

// Stdout returns the captured task stdout stream.
func (u *UI) Stdout() io.Writer {
	return u.stdout
}

// Stderr returns the captured task stderr stream.
func (u *UI) Stderr() io.Writer {
	return u.stderr
}

// Run runs a normal progress function with captured task streams.
func (u *UI) Run(fn Func) error {
	if fn == nil {
		return errors.New("progress UI run function is nil")
	}
	return fn(u.stdin, u.stdout, u.stderr)
}

// Takeover temporarily gives the callback direct control of stdout and stderr.
//
// Captured output written before and after the takeover keeps the usual
// progress behavior. Takeover is only available for sequential tasks.
func (u *UI) Takeover(fn Func) error {
	if fn == nil {
		return errors.New("progress UI takeover function is nil")
	}
	if u.parallelRun {
		return errors.New("progress UI takeover is not available in parallel tasks")
	}
	if u.renderer != nil {
		u.renderer.suspendSingle(u.taskName)
		defer u.renderer.resumeSingle(u.taskName)
	}
	return fn(u.stdin, u.rawStdout, u.rawStderr)
}
