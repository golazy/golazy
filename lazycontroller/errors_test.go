package lazycontroller

import (
	"errors"
	"strings"
	"testing"

	"golazy.dev/lazyerrors"
)

func TestPanicErrorExposesBacktrace(t *testing.T) {
	err := recoveredPanicError()
	if err == nil {
		t.Fatal("PanicError returned nil")
	}
	if got, want := err.Error(), "panic: boom"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}

	var traced interface {
		Backtrace() []lazyerrors.Frame
	}
	if !errors.As(err, &traced) {
		t.Fatalf("error does not implement Backtrace() []lazyerrors.Frame")
	}
	frames := traced.Backtrace()
	if !containsFunction(frames, "panicSource") {
		t.Fatalf("backtrace = %#v, want panicSource", frames)
	}

	frames[0] = lazyerrors.Frame{Function: "changed"}
	if got := traced.Backtrace()[0].Function; got == "changed" {
		t.Fatal("Backtrace() returned mutable internal storage")
	}
}

func recoveredPanicError() (err error) {
	defer func() {
		err = PanicError(recover())
	}()
	panicSource()
	return nil
}

//go:noinline
func panicSource() {
	panic("boom")
}

func containsFunction(frames []lazyerrors.Frame, name string) bool {
	for _, frame := range frames {
		if strings.Contains(frame.Function, name) {
			return true
		}
	}
	return false
}
