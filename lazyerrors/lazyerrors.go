package lazyerrors

import (
	"fmt"
	"runtime"
	"strings"
)

const maxBacktraceFrames = 64

// Frame is one recorded application backtrace frame.
type Frame struct {
	Function string
	File     string
	Line     int
}

func (f Frame) String() string {
	switch {
	case f.Function == "" && f.File == "":
		return ""
	case f.Function == "":
		return fmt.Sprintf("%s:%d", f.File, f.Line)
	case f.File == "":
		return f.Function
	default:
		return fmt.Sprintf("%s %s:%d", f.Function, f.File, f.Line)
	}
}

type lazyError struct {
	prefix string
	err    error
	frames []Frame
}

// New formats an error like fmt.Errorf, prefixes it with the caller name, and
// records a backtrace beginning at the caller.
func New(format string, args ...any) error {
	formatted := fmt.Errorf(format, args...)
	frames := captureFrames(3)

	base := &lazyError{
		prefix: callerPrefix(frames),
		err:    formatted,
		frames: frames,
	}

	if unwrapper, ok := formatted.(interface{ Unwrap() []error }); ok {
		return &multiWrapError{base: base, causes: unwrapper.Unwrap()}
	}
	if unwrapper, ok := formatted.(interface{ Unwrap() error }); ok {
		return &wrapError{base: base, cause: unwrapper.Unwrap()}
	}
	return base
}

func (e *lazyError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.err == nil {
		return "<nil>"
	}
	if e.prefix == "" {
		return e.err.Error()
	}
	return e.prefix + ": " + e.err.Error()
}

func (e *lazyError) Backtrace() []Frame {
	if e == nil || len(e.frames) == 0 {
		return nil
	}
	return append([]Frame(nil), e.frames...)
}

type wrapError struct {
	base  *lazyError
	cause error
}

func (e *wrapError) Error() string {
	return e.base.Error()
}

func (e *wrapError) Backtrace() []Frame {
	return e.base.Backtrace()
}

func (e *wrapError) Unwrap() error {
	return e.cause
}

type multiWrapError struct {
	base   *lazyError
	causes []error
}

func (e *multiWrapError) Error() string {
	return e.base.Error()
}

func (e *multiWrapError) Backtrace() []Frame {
	return e.base.Backtrace()
}

func (e *multiWrapError) Unwrap() []error {
	return append([]error(nil), e.causes...)
}

func captureFrames(skip int) []Frame {
	var pcs [maxBacktraceFrames]uintptr
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return nil
	}

	runtimeFrames := runtime.CallersFrames(pcs[:n])
	frames := make([]Frame, 0, n)
	for {
		runtimeFrame, more := runtimeFrames.Next()
		frames = append(frames, Frame{
			Function: shortFuncName(runtimeFrame.Function),
			File:     runtimeFrame.File,
			Line:     runtimeFrame.Line,
		})
		if !more {
			break
		}
	}
	return frames
}

func callerPrefix(frames []Frame) string {
	if len(frames) == 0 {
		return ""
	}
	return frames[0].Function
}

func shortFuncName(name string) string {
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
