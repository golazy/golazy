package lazycontroller

import (
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"golazy.dev/lazyerrors"
)

const maxPanicBacktraceFrames = 64

type HTTPError struct {
	Status int
	Err    error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d %s: %v", e.Status, http.StatusText(e.Status), e.Err)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func Error(status int, err error) error {
	return &HTTPError{Status: status, Err: err}
}

func StatusCode(err error) int {
	status := http.StatusInternalServerError
	var httpError *HTTPError
	if errors.As(err, &httpError) && httpError.Status != 0 {
		status = httpError.Status
	}
	return status
}

func PanicError(recovered any) error {
	message := ""
	switch value := recovered.(type) {
	case nil:
		return nil
	case error:
		message = fmt.Sprintf("panic: %v", value)
	default:
		message = fmt.Sprintf("panic: %v", value)
	}
	return &panicError{message: message, frames: capturePanicFrames(3)}
}

type panicError struct {
	message string
	frames  []lazyerrors.Frame
}

func (e *panicError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.message
}

func (e *panicError) Backtrace() []lazyerrors.Frame {
	if e == nil || len(e.frames) == 0 {
		return nil
	}
	return append([]lazyerrors.Frame(nil), e.frames...)
}

func capturePanicFrames(skip int) []lazyerrors.Frame {
	var pcs [maxPanicBacktraceFrames]uintptr
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return nil
	}

	runtimeFrames := runtime.CallersFrames(pcs[:n])
	frames := make([]lazyerrors.Frame, 0, n)
	for {
		runtimeFrame, more := runtimeFrames.Next()
		frames = append(frames, lazyerrors.Frame{
			Function: shortPanicFuncName(runtimeFrame.Function),
			File:     runtimeFrame.File,
			Line:     runtimeFrame.Line,
		})
		if !more {
			break
		}
	}
	return frames
}

func shortPanicFuncName(name string) string {
	if name == "" {
		return ""
	}
	if slash := strings.LastIndex(name, "/"); slash >= 0 {
		name = name[slash+1:]
	}
	name = strings.ReplaceAll(name, "(*", "")
	name = strings.ReplaceAll(name, ")", "")
	return name
}
