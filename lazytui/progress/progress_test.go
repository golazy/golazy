package progress

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestNewRunsTasksAndHidesSuccessfulOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		Task("one", func(_ io.Reader, out io.Writer, errout io.Writer) error {
			fmt.Fprintln(out, "hidden stdout")
			fmt.Fprintln(errout, "hidden stderr")
			return nil
		}),
		Task("two", func(_ io.Reader, out io.Writer, errout io.Writer) error {
			fmt.Fprintln(out, "also hidden")
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := stdout.String(), "* one ... DONE\n* two ... DONE\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestNewStopsOnErrorAndFlushesCapturedOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	boom := errors.New("boom")
	called := false

	err := run(Tasks{
		Task("fail", func(_ io.Reader, out io.Writer, errout io.Writer) error {
			fmt.Fprintln(out, "stdout detail")
			fmt.Fprintln(errout, "stderr detail")
			return boom
		}),
		Task("later", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
			called = true
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})

	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom", err)
	}
	if called {
		t.Fatal("later task ran after non-warning error")
	}
	if got, want := stdout.String(), "* fail ... ERROR\nstdout detail\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "stderr detail\n  error: boom\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestNewContinuesAfterWarning(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	warning := errors.New("keep going")
	called := false

	err := run(Tasks{
		Task("warn", func(_ io.Reader, out io.Writer, _ io.Writer) error {
			fmt.Fprintln(out, "warn detail")
			return Warn{Err: warning}
		}),
		Task("next", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
			called = true
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("next task did not run after warning")
	}
	if got, want := stdout.String(), "* warn ... WARN\nwarn detail\n* next ... DONE\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "  warning: keep going\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestUITaskCapturesOutputOnSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		UITask("ui", func(ui *UI) error {
			fmt.Fprintln(ui.Stdout(), "hidden stdout")
			fmt.Fprintln(ui.Stderr(), "hidden stderr")
			return ui.Run(func(_ io.Reader, out io.Writer, errout io.Writer) error {
				fmt.Fprintln(out, "also hidden")
				fmt.Fprintln(errout, "still hidden")
				return nil
			})
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := stdout.String(), "* ui ... DONE\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestUITaskTakeoverUsesRawStreamsAndResumesProgress(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		UITask("review", func(ui *UI) error {
			fmt.Fprintln(ui.Stdout(), "captured before")
			if err := ui.Takeover(func(_ io.Reader, out io.Writer, errout io.Writer) error {
				fmt.Fprintln(out, "visible stdout")
				fmt.Fprintln(errout, "visible stderr")
				return nil
			}); err != nil {
				return err
			}
			fmt.Fprintln(ui.Stdout(), "captured after")
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := stdout.String(), "* review ...\nvisible stdout\n* review ... DONE\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "visible stderr\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestUITaskTakeoverFlushesCapturedOutputOnError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	boom := errors.New("boom")

	err := run(Tasks{
		UITask("review", func(ui *UI) error {
			fmt.Fprintln(ui.Stdout(), "captured stdout")
			fmt.Fprintln(ui.Stderr(), "captured stderr")
			return ui.Takeover(func(_ io.Reader, out io.Writer, errout io.Writer) error {
				fmt.Fprintln(out, "visible stdout")
				fmt.Fprintln(errout, "visible stderr")
				return boom
			})
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if !errors.Is(err, boom) {
		t.Fatalf("error = %v, want boom", err)
	}

	if got, want := stdout.String(), "* review ...\nvisible stdout\n* review ... ERROR\ncaptured stdout\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "visible stderr\ncaptured stderr\n  error: boom\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestUITaskTakeoverIsRejectedInParallel(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		Parallel("group", Tasks{
			UITask("ui", func(ui *UI) error {
				return ui.Takeover(func(_ io.Reader, _ io.Writer, _ io.Writer) error {
					return nil
				})
			}),
		}),
	}, environment{stdout: &stdout, stderr: &stderr})
	if err == nil {
		t.Fatal("err = nil, want takeover rejection")
	}
	if !strings.Contains(err.Error(), "takeover is not available in parallel tasks") {
		t.Fatalf("error = %v", err)
	}
	if got, want := stdout.String(), "* ui ... Working\n* ui ... ERROR\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "  error: progress UI takeover is not available in parallel tasks\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestWarnUnwrapsUnderlyingError(t *testing.T) {
	wrapped := &customWarningError{}
	err := Warn{Err: wrapped}

	var found *customWarningError
	if !errors.As(err, &found) {
		t.Fatal("Warn did not unwrap underlying error")
	}
	if found != wrapped {
		t.Fatalf("found = %p, want %p", found, wrapped)
	}
}

type customWarningError struct{}

func (*customWarningError) Error() string { return "custom warning" }

func TestParallelRunsTasksConcurrently(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	started := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- run(Tasks{
			Parallel("group", Tasks{
				Task("one", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
					started <- "one"
					<-release
					return nil
				}),
				Task("two", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
					started <- "two"
					<-release
					return nil
				}),
			}),
		}, environment{stdout: &stdout, stderr: &stderr})
	}()

	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case name := <-started:
			seen[name] = true
		case <-time.After(time.Second):
			t.Fatal("parallel tasks did not both start before release")
		}
	}
	close(release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("parallel tasks did not finish")
	}

	got := stdout.String()
	for _, want := range []string{
		"* one ... Working\n",
		"* two ... Working\n",
		"* one ... DONE\n",
		"* two ... DONE\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, missing %q", got, want)
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestParallelReturnsJoinedFailuresAfterAllTasksFinish(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	first := errors.New("first")
	second := errors.New("second")
	warning := errors.New("warning")
	called := false

	err := run(Tasks{
		Parallel("group", Tasks{
			Task("first", func(_ io.Reader, out io.Writer, _ io.Writer) error {
				fmt.Fprintln(out, "first detail")
				return first
			}),
			Task("warn", func(_ io.Reader, _ io.Writer, errout io.Writer) error {
				fmt.Fprintln(errout, "warn detail")
				return Warn{Err: warning}
			}),
			Task("second", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
				return second
			}),
		}),
		Task("after", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
			called = true
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})

	if !errors.Is(err, first) || !errors.Is(err, second) {
		t.Fatalf("error = %v, want joined first and second", err)
	}
	if errors.Is(err, warning) {
		t.Fatalf("error = %v, should not join warning into failure", err)
	}
	if called {
		t.Fatal("task after failing parallel group ran")
	}

	gotOut := stdout.String()
	for _, want := range []string{
		"* first ... ERROR\n",
		"* warn ... WARN\n",
		"* second ... ERROR\n",
		"first detail\n",
	} {
		if !strings.Contains(gotOut, want) {
			t.Fatalf("stdout = %q, missing %q", gotOut, want)
		}
	}
	gotErr := stderr.String()
	for _, want := range []string{
		"warn detail\n",
		"  error: first\n",
		"  warning: warning\n",
		"  error: second\n",
	} {
		if !strings.Contains(gotErr, want) {
			t.Fatalf("stderr = %q, missing %q", gotErr, want)
		}
	}
}

func TestParallelWarningOnlyGroupContinues(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	called := false

	err := run(Tasks{
		Parallel("group", Tasks{
			Task("warn", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
				return Warn{Err: errors.New("soft")}
			}),
		}),
		Task("after", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
			called = true
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr})

	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("task after warning-only parallel group did not run")
	}
	if got := stdout.String(); !strings.Contains(got, "* after ... DONE\n") {
		t.Fatalf("stdout = %q, missing after task", got)
	}
}

func TestInteractiveSingleUsesRewriteAndColor(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		Task("one", func(_ io.Reader, _ io.Writer, _ io.Writer) error {
			return nil
		}),
	}, environment{stdout: &stdout, stderr: &stderr, interactive: true, color: true})
	if err != nil {
		t.Fatal(err)
	}

	got := stdout.String()
	for _, want := range []string{"\r\x1b[2K", ansiCyan, ansiTeal} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, missing %q", got, want)
		}
	}
}

func TestInteractiveParallelUsesMultipleLines(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run(Tasks{
		Parallel("group", Tasks{
			Task("one", func(_ io.Reader, _ io.Writer, _ io.Writer) error { return nil }),
			Task("two", func(_ io.Reader, _ io.Writer, _ io.Writer) error { return nil }),
		}),
	}, environment{stdout: &stdout, stderr: &stderr, interactive: true})
	if err != nil {
		t.Fatal(err)
	}

	got := stdout.String()
	for _, want := range []string{
		"* one Working\n",
		"* two Working\n",
		"\x1b[2A",
		"\x1b[1A",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, missing %q", got, want)
		}
	}
}

func TestSupportsColor(t *testing.T) {
	original := getenv
	t.Cleanup(func() { getenv = original })

	tests := []struct {
		name        string
		interactive bool
		env         map[string]string
		want        bool
	}{
		{name: "interactive", interactive: true, env: map[string]string{"TERM": "xterm-256color"}, want: true},
		{name: "noninteractive", interactive: false, env: map[string]string{"TERM": "xterm-256color"}, want: false},
		{name: "no color", interactive: true, env: map[string]string{"TERM": "xterm-256color", "NO_COLOR": "1"}, want: false},
		{name: "dumb", interactive: true, env: map[string]string{"TERM": "dumb"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv = func(key string) string {
				return tt.env[key]
			}
			if got := supportsColor(tt.interactive); got != tt.want {
				t.Fatalf("supportsColor() = %v, want %v", got, tt.want)
			}
		})
	}
}
