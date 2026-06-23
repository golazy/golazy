package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

const (
	ansiClearLine = "\x1b[2K"
	ansiReset     = "\x1b[0m"
	ansiCyan      = "\x1b[38;2;0;173;216m"
	ansiTeal      = "\x1b[38;2;0;162;156m"
	ansiYellow    = "\x1b[38;2;253;221;0m"
	ansiMagenta   = "\x1b[38;2;206;50;98m"
)

type renderer struct {
	mu          sync.Mutex
	stdout      io.Writer
	stderr      io.Writer
	interactive bool
	color       bool
}

func newRenderer(env environment) *renderer {
	return &renderer{
		stdout:      env.stdout,
		stderr:      env.stderr,
		interactive: env.interactive,
		color:       env.color,
	}
}

func (r *renderer) startSingle(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.interactive {
		fmt.Fprintf(r.stdout, "* %s %s", name, r.decorate("Working", statusDone))
		return
	}
	fmt.Fprintf(r.stdout, "* %s ...", name)
}

func (r *renderer) finishSingle(name string, status taskStatus, stdout string, stderr string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.interactive {
		fmt.Fprintf(r.stdout, "\r%s* %s %s\n", ansiClearLine, name, r.decorate(string(status), status))
	} else {
		fmt.Fprintf(r.stdout, " %s\n", status)
	}
	r.flushFailureLocked(stdout, stderr, err)
}

func (r *renderer) startParallel(tasks Tasks) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, task := range tasks {
		if r.interactive {
			fmt.Fprintf(r.stdout, "* %s %s\n", task.name, r.decorate("Working", statusDone))
			continue
		}
		fmt.Fprintf(r.stdout, "* %s ... Working\n", task.name)
	}
}

func (r *renderer) finishParallel(index int, name string, status taskStatus, total int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.interactive {
		fmt.Fprintf(r.stdout, "* %s ... %s\n", name, status)
		return
	}

	move := total - index
	if move > 0 {
		fmt.Fprintf(r.stdout, "\x1b[%dA", move)
	}
	fmt.Fprintf(r.stdout, "\r%s* %s %s", ansiClearLine, name, r.decorate(string(status), status))
	if move > 0 {
		fmt.Fprintf(r.stdout, "\x1b[%dB\r", move)
	}
}

func (r *renderer) flushFailure(stdout string, stderr string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flushFailureLocked(stdout, stderr, err)
}

func (r *renderer) flushFailureLocked(stdout string, stderr string, err error) {
	if err == nil {
		return
	}
	writeCaptured(r.stdout, stdout)
	writeCaptured(r.stderr, stderr)
	label := "error"
	if isWarning(err) {
		label = "warning"
	}
	fmt.Fprintf(r.stderr, "  %s: %v\n", label, err)
}

func writeCaptured(writer io.Writer, value string) {
	if value == "" {
		return
	}
	fmt.Fprint(writer, value)
	if !strings.HasSuffix(value, "\n") {
		fmt.Fprintln(writer)
	}
}

func (r *renderer) decorate(value string, status taskStatus) string {
	if !r.color {
		return value
	}
	color := ansiTeal
	switch status {
	case statusWarn:
		color = ansiYellow
	case statusError:
		color = ansiMagenta
	case statusDone:
		if value == "Working" {
			color = ansiCyan
		}
	}
	return color + value + ansiReset
}
