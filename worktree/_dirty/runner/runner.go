// Package runner run a restart a program on signals
package runner

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// Options holds the runner options
type Options struct {
	KillWaitPeriod time.Duration // Time wait for a proces to die before callking kill
	ReadyString    []string      // ReadyString is the string that Runner looks for in the command output to send EventReady
}

// DefaultRunnerOptions are used when no RunnerOptions are passed
var DefaultRunnerOptions = &Options{
	KillWaitPeriod: time.Second,
	ReadyString:    []string{"Listening", "Started", "Ready"},
}

// EventStart is fired whenever Start is called
type EventStart struct {
	// Err display any error that happen during start
	// If Err is nil you can expect an EventStarted event
	Err error
}

// EventStop is fired whenever Stop is called
type EventStop struct {
	// Err display any error that happen during stop
	Err error
}

// EventSignal is fired whenever
type EventSignal struct {
	// Err display any error that happen during stop
	Err error
}

// EventReady is fired whenever the command outputs the string Listening
type EventReady struct {
	Data string // Data contains the block that triggered the EventReady
}

// EventRestart is fired whenever Restart is called
type EventRestart struct {
	// Err display any error that happen during stop
	// If Err is nil you can expect an EventStarted event
	Err error
}

// EventStopped is fired whenever the process exits
type EventStopped struct {
	Output   []string // Holds the command output up to MaxOutputSize
	ExitCode int      // ExitCode holds the exit code
	RunTime  time.Duration
}

// EventStarted is fired whenever the subprocess is started
type EventStarted struct {
	Command string
}

// Runner is an command runner that produces events on start/stop and restart
type Runner struct {
	Events     <-chan (any) // Events will be fired here. The channel is not expected to be closed.
	options    Options
	cmd        *exec.Cmd
	e          chan (any)
	startCmd   chan (chan (error))
	restartCmd chan (chan (error))
	stopCmd    chan (chan (error))
	closeCmd   chan (chan (error))
	signalCmd  chan (struct {
		Signal syscall.Signal
		errC   chan (error)
	})
	closed bool
}

// Close stop all the internal goroutines
// After Close is called the runner can't be used anymore
func (r *Runner) Close() error {
	if r.closed {
		return nil
	}
	errC := make(chan (error))
	r.closeCmd <- errC
	return <-errC
}

// Start starts the command
// If the command is already running it returns ErrRunning
func (r *Runner) Start() error {
	if r.closed {
		return ErrRunnerClosed
	}
	errC := make(chan (error))
	r.startCmd <- errC
	return <-errC
}

// Restart restart the process by calling Stop and then Restart. If the process is not runing it will be the same as calling Start
func (r *Runner) Restart() error {
	if r.closed {
		return ErrRunnerClosed
	}
	errC := make(chan (error))
	r.restartCmd <- errC
	return <-errC
}

// Stop stops the process.
// It will send an interrupt signal to the process.
// If after KillWaitPeriod the process is still alive, it will send a kill signal
func (r *Runner) Stop() error {
	if r.closed {
		return ErrRunnerClosed
	}
	errC := make(chan (error))
	r.stopCmd <- errC
	return <-errC
}

// Signal sends a signal to the process.
// If the process is not running it returns ErrNotRunning
func (r *Runner) Signal(s syscall.Signal) error {
	if r.closed {
		return ErrRunnerClosed
	}
	errC := make(chan (error))
	r.signalCmd <- struct {
		Signal syscall.Signal
		errC   chan error
	}{s, errC}
	return <-errC
}

// New creates a new runner for the given command
// if options is nil, New will use DefaultRunnerOptions
func New(cmd *exec.Cmd, options *Options) *Runner {

	e := make(chan (any), 1024)
	if options == nil {
		options = DefaultRunnerOptions
	}
	r := &Runner{
		Events:     e,
		options:    *options,
		cmd:        cmd,
		e:          e,
		startCmd:   make(chan (chan (error))),
		restartCmd: make(chan (chan (error))),
		stopCmd:    make(chan (chan (error))),
		closeCmd:   make(chan (chan (error))),
		closed:     false,
	}
	go r.loop()
	return r
}

var (
	// ErrRunning is the return error in the Start method
	ErrRunning = errors.New("Program is already running")
	// ErrCantKill is returned by Restart and Stop in case the process can't be killed
	ErrCantKill = errors.New("Process is still alive after sending the kill signal")
	// ErrNotRunning is retuned by the Stop and Signal command when the program is not running
	ErrNotRunning = errors.New("Process is not running")
	// ErrRunnerClosed is returned by any method when the runner is closed
	ErrRunnerClosed = errors.New("Runner is closed")
)

func (r *Runner) loop() {
	running := false
	var done chan (int) = nil
	var io chan ([]byte) = nil
	var readyEventSent bool
	var output []string
	var startTime time.Time
	var cmd *exec.Cmd

	signal := func(sig syscall.Signal) error {
		if cmd == nil || cmd.Process == nil {
			return ErrNotRunning
		}
		syscall.Kill(-cmd.Process.Pid, sig)
		err := cmd.Process.Signal(sig)
		return err

	}

	start := func() error {
		startTime = time.Now()
		io = make(chan ([]byte))
		cmd = &exec.Cmd{
			Path:        r.cmd.Path,
			Args:        r.cmd.Args,
			Env:         r.cmd.Env,
			Dir:         r.cmd.Dir,
			Stdin:       r.cmd.Stdin,
			Stdout:      channelWriter(io),
			Stderr:      channelWriter(io),
			ExtraFiles:  r.cmd.ExtraFiles,
			SysProcAttr: &syscall.SysProcAttr{Setpgid: true},
		}

		output = make([]string, 0, 1024)
		readyEventSent = false

		if err := cmd.Start(); err != nil {
			return err
		}
		running = true
		r.e <- EventStarted{
			Command: strings.Join(append([]string{cmd.Path}, cmd.Args...), " "),
		}

		// Wait for it to stop
		done = make(chan (int))
		go func() {
			err := cmd.Wait()
			if exitError, ok := err.(*exec.ExitError); ok {
				done <- exitError.ExitCode()
				return
			}
			done <- cmd.ProcessState.ExitCode()
		}()

		return nil
	}

	checkReadyEvent := func(data []byte) {
		if !readyEventSent {
			for _, readyString := range r.options.ReadyString {
				if bytes.Contains(data, []byte(readyString)) {
					r.e <- EventReady{string(data)}
					readyEventSent = true
				}
			}
		}
	}

	buf := []byte{}
	processIO := func(data []byte) {
		if r.cmd.Stdout != nil {
			r.cmd.Stdout.Write(data)
		}
		buf := append(buf, data...)
		if len(buf) == 0 {
			return
		}
		checkReadyEvent(buf)
		lines := strings.Split(string(buf), "\n")
		output = append(output, lines[0:len(lines)-1]...)
		buf = []byte(lines[len(lines)-1])
	}

	handleExit := func(statusCode int) {
		if len(buf) != 0 {
			output = append(output, string(buf))
		}
		running = false
		done = nil
		r.e <- EventStopped{
			Output:   output,
			ExitCode: statusCode,
			RunTime:  time.Since(startTime),
		}
	}

	stop := func() error {
		signal(syscall.SIGINT)

		wait := time.After(time.Duration(r.options.KillWaitPeriod))
		var kill <-chan (time.Time) = nil

		for running {
			select {
			case exitCode := <-done:
				handleExit(exitCode)
				return nil
			case data := <-io:
				processIO(data)
			case <-wait:
				signal(syscall.SIGKILL)
				kill = time.After(time.Duration(r.options.KillWaitPeriod))
			case <-kill:
				return ErrCantKill
			}
		}
		return nil
	}

	for {
		select {
		case errC := <-r.startCmd:
			if running {
				errC <- ErrRunning
				r.e <- EventStart{Err: ErrRunning}
				continue
			}
			err := start()
			r.e <- EventStart{Err: err}
			errC <- err

		case errC := <-r.restartCmd:
			if running {
				err := stop()
				if err != nil {
					r.e <- EventRestart{Err: err}
					errC <- err
					continue
				}
			}
			err := start()
			r.e <- EventRestart{Err: err}
			errC <- err
		case errC := <-r.stopCmd:
			if !running {
				errC <- ErrNotRunning
				r.e <- EventStop{Err: ErrNotRunning}
				continue
			}
			err := stop()
			r.e <- EventStop{Err: err}
			errC <- err

		case args := <-r.signalCmd:
			args.errC <- signal(args.Signal)
		case errC := <-r.closeCmd:
			if running {
				errC <- stop()
			}
			errC <- nil
			return
		case exitCode := <-done:
			handleExit(exitCode)
		case data := <-io:
			processIO(data)
		}
	}
}

type channelWriter chan ([]byte)

func (cw channelWriter) Write(data []byte) (int, error) {
	cw <- data
	return len(data), nil
}
