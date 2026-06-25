package lazyerrors_test

import (
	"errors"
	"strings"
	"testing"

	"golazy.dev/lazyerrors"
)

func TestNewPrefixesFunctionCaller(t *testing.T) {
	err := functionError()

	if got, want := err.Error(), "lazyerrors_test.functionError: failed to load"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNewPrefixesMethodCaller(t *testing.T) {
	err := (&worker{}).methodError()

	if got, want := err.Error(), "lazyerrors_test.worker.methodError: missing value"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestBacktraceStartsAtCaller(t *testing.T) {
	err := functionError()

	backtrace := testBacktraceOf(t, err)
	if len(backtrace) == 0 {
		t.Fatal("Backtrace() returned no frames")
	}
	if !strings.Contains(backtrace[0].Function, "lazyerrors_test.functionError") {
		t.Fatalf("first backtrace frame = %#v, want functionError", backtrace[0])
	}
	if !strings.Contains(backtrace[0].String(), "lazyerrors_test.functionError") {
		t.Fatalf("first backtrace frame string = %q, want functionError", backtrace[0].String())
	}

	backtrace[0] = lazyerrors.Frame{Function: "changed"}
	if got := testBacktraceOf(t, err)[0]; got.Function == "changed" {
		t.Fatal("Backtrace() returned mutable internal storage")
	}
}

func TestFrameString(t *testing.T) {
	tests := []struct {
		name  string
		frame lazyerrors.Frame
		want  string
	}{
		{
			name:  "function file and line",
			frame: lazyerrors.Frame{Function: "posts.Controller.Show", File: "/app/posts.go", Line: 42},
			want:  "posts.Controller.Show /app/posts.go:42",
		},
		{
			name:  "file and line",
			frame: lazyerrors.Frame{File: "/app/posts.go", Line: 42},
			want:  "/app/posts.go:42",
		},
		{
			name:  "function only",
			frame: lazyerrors.Frame{Function: "posts.Controller.Show"},
			want:  "posts.Controller.Show",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.frame.String(); got != test.want {
				t.Fatalf("String() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNewPreservesSingleUnwrap(t *testing.T) {
	cause := errors.New("root cause")

	err := wrappedError(cause)

	if got := errors.Unwrap(err); got != cause {
		t.Fatalf("errors.Unwrap(err) = %v, want root cause", got)
	}
	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(err, cause) = false")
	}
	if got, want := err.Error(), "lazyerrors_test.wrappedError: load record: root cause"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestNewPreservesMultiUnwrap(t *testing.T) {
	first := errors.New("first")
	second := errors.New("second")

	err := multiWrappedError(first, second)

	if got := errors.Unwrap(err); got != nil {
		t.Fatalf("errors.Unwrap(err) = %v, want nil for multi-unwrapper", got)
	}

	unwrapper, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatal("error does not implement Unwrap() []error")
	}
	causes := unwrapper.Unwrap()
	if len(causes) != 2 || causes[0] != first || causes[1] != second {
		t.Fatalf("Unwrap() = %#v, want first and second", causes)
	}

	causes[0] = nil
	if got := unwrapper.Unwrap()[0]; got != first {
		t.Fatal("Unwrap() []error returned mutable internal storage")
	}

	if !errors.Is(err, first) {
		t.Fatal("errors.Is(err, first) = false")
	}
	if !errors.Is(err, second) {
		t.Fatal("errors.Is(err, second) = false")
	}
}

//go:noinline
func functionError() error {
	return lazyerrors.New("failed to load")
}

type worker struct{}

//go:noinline
func (w *worker) methodError() error {
	return lazyerrors.New("missing value")
}

//go:noinline
func wrappedError(cause error) error {
	return lazyerrors.New("load record: %w", cause)
}

//go:noinline
func multiWrappedError(first, second error) error {
	return lazyerrors.New("combine: %w; %w", first, second)
}

func testBacktraceOf(t *testing.T, err error) []lazyerrors.Frame {
	t.Helper()

	var traced interface {
		Backtrace() []lazyerrors.Frame
	}
	if !errors.As(err, &traced) {
		t.Fatal("error does not implement Backtrace() []lazyerrors.Frame")
	}
	return traced.Backtrace()
}
