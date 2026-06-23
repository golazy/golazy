package progress

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// Func is the function shape run by a progress task.
type Func func(stdin io.Reader, stdout io.Writer, stderr io.Writer) error

// TaskDefinition is an opaque progress task created by Task, UITask, or
// Parallel.
type TaskDefinition struct {
	name     string
	run      Func
	uiRun    func(*UI) error
	parallel Tasks
}

// Tasks is a list of task definitions run by New.
type Tasks []TaskDefinition

// Warn marks an error as warning-only.
//
// New continues after a Warn, while still printing WARN and any captured task
// output. Unwrap exposes the underlying error to errors.Is and errors.As.
type Warn struct {
	Err error
}

func (w Warn) Error() string {
	if w.Err == nil {
		return "warning"
	}
	return w.Err.Error()
}

func (w Warn) Unwrap() error {
	return w.Err
}

// Task creates a named progress task.
func Task(name string, run Func) TaskDefinition {
	return TaskDefinition{name: name, run: run}
}

// UITask creates a named progress task that can temporarily take over the
// terminal for interactive work.
func UITask(name string, run func(*UI) error) TaskDefinition {
	return TaskDefinition{name: name, uiRun: run}
}

// Parallel creates a named task that runs child tasks concurrently.
//
// The returned value is a normal TaskDefinition and can be placed inside Tasks
// with sequential tasks.
func Parallel(name string, tasks Tasks) TaskDefinition {
	copied := make(Tasks, len(tasks))
	copy(copied, tasks)
	return TaskDefinition{name: name, parallel: copied}
}

// New runs tasks with process standard streams until a task returns a
// non-warning error.
func New(tasks Tasks) error {
	return Run(tasks, os.Stdin, os.Stdout, os.Stderr)
}

// Run runs tasks with explicit streams until a task returns a non-warning
// error. It is useful for command packages that already receive stdin, stdout,
// and stderr from their caller.
func Run(tasks Tasks, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	interactive := isTerminal(stdout)
	return run(tasks, environment{
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
		interactive: interactive,
		color:       supportsColor(interactive),
	})
}

func run(tasks Tasks, env environment) error {
	env = normalizeEnvironment(env)
	runner := &runner{
		env:      env,
		renderer: newRenderer(env),
	}
	for _, task := range tasks {
		err := runner.runTask(task)
		if err == nil || isWarning(err) {
			continue
		}
		return err
	}
	return nil
}

type runner struct {
	env      environment
	renderer *renderer
}

func (r *runner) runTask(task TaskDefinition) error {
	if task.parallel != nil {
		return r.runParallel(task)
	}
	return r.runSingle(task)
}

func (r *runner) runSingle(task TaskDefinition) error {
	r.renderer.startSingle(task.name)
	stdout := &captureBuffer{}
	stderr := &captureBuffer{}
	err := r.runTaskFunc(task, stdout, stderr, false)
	r.renderer.finishSingle(task.name, statusFor(err), stdout.String(), stderr.String(), err)
	return err
}

func (r *runner) runParallel(task TaskDefinition) error {
	tasks := task.parallel
	if len(tasks) == 0 {
		return nil
	}

	r.renderer.startParallel(tasks)
	results := make([]parallelResult, len(tasks))
	var wg sync.WaitGroup
	for index, child := range tasks {
		wg.Add(1)
		go func(index int, child TaskDefinition) {
			defer wg.Done()
			stdout := &captureBuffer{}
			stderr := &captureBuffer{}
			err := r.runTaskFunc(child, stdout, stderr, true)
			results[index] = parallelResult{
				name:   child.name,
				stdout: stdout.String(),
				stderr: stderr.String(),
				err:    err,
			}
			r.renderer.finishParallel(index, child.name, statusFor(err), len(tasks))
		}(index, child)
	}
	wg.Wait()

	var warnings []error
	var failures []error
	for _, result := range results {
		if result.err == nil {
			continue
		}
		r.renderer.flushFailure(result.stdout, result.stderr, result.err)
		if isWarning(result.err) {
			warnings = append(warnings, result.err)
			continue
		}
		failures = append(failures, result.err)
	}
	if len(failures) != 0 {
		return errors.Join(failures...)
	}
	if len(warnings) != 0 {
		return Warn{Err: errors.Join(warnings...)}
	}
	return nil
}

type parallelResult struct {
	name   string
	stdout string
	stderr string
	err    error
}

func (r *runner) runTaskFunc(task TaskDefinition, stdout io.Writer, stderr io.Writer, parallel bool) error {
	if task.parallel != nil {
		return fmt.Errorf("%s: nested parallel tasks are not supported", task.name)
	}
	if task.uiRun != nil {
		return task.uiRun(&UI{
			stdin:       r.env.stdin,
			stdout:      stdout,
			stderr:      stderr,
			rawStdout:   r.env.stdout,
			rawStderr:   r.env.stderr,
			renderer:    r.renderer,
			taskName:    task.name,
			parallelRun: parallel,
		})
	}
	if task.run == nil {
		return fmt.Errorf("%s: task function is nil", task.name)
	}
	return task.run(r.env.stdin, stdout, stderr)
}

type taskStatus string

const (
	statusDone  taskStatus = "DONE"
	statusWarn  taskStatus = "WARN"
	statusError taskStatus = "ERROR"
)

func statusFor(err error) taskStatus {
	if err == nil {
		return statusDone
	}
	if isWarning(err) {
		return statusWarn
	}
	return statusError
}

func isWarning(err error) bool {
	if err == nil {
		return false
	}
	var warn Warn
	if errors.As(err, &warn) {
		return true
	}
	var warnPointer *Warn
	return errors.As(err, &warnPointer)
}

type captureBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *captureBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *captureBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type environment struct {
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
	interactive bool
	color       bool
}

func normalizeEnvironment(env environment) environment {
	if env.stdin == nil {
		env.stdin = bytes.NewReader(nil)
	}
	if env.stdout == nil {
		env.stdout = io.Discard
	}
	if env.stderr == nil {
		env.stderr = io.Discard
	}
	return env
}

func defaultEnvironment() environment {
	interactive := isTerminal(os.Stdout)
	return environment{
		stdin:       os.Stdin,
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		interactive: interactive,
		color:       supportsColor(interactive),
	}
}
