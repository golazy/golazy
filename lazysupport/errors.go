package lazysupport

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
)

// ErrorWithStack is an error with a stacktrace attached
type ErrorWithStack struct {
	offset     int
	Err        error
	stacktrace []StackLine
	stack      []byte
}

// Error calls Error in the underlying error
func (e ErrorWithStack) Error() string {
	return e.Err.Error()
}

// Unwrap calls Unwrap in the underlying error
func (e ErrorWithStack) Unwrap() error {
	return e.Err
}

// Trace returns the stacktrace
func (e ErrorWithStack) Stacktrace() *Stacktrace {
	if len(e.stacktrace) == 0 {
		if len(e.stack) == 0 {
			return nil
		}
		e.stacktrace = StackDecode(e.stack)
		if e.offset > 0 {
			e.stacktrace = e.stacktrace[min(e.offset, len(e.stacktrace)):]
		}

	}
	return &Stacktrace{StackLines: e.stacktrace}
}

// String returns the error message and the stacktrace
func (e ErrorWithStack) String() string {
	buf := "err: " + e.Err.Error() + "\n"
	buf += e.Stacktrace().String()
	return buf
}

func NewErrorf(offset int, format string, data ...any) *ErrorWithStack {
	return NewError(offset, fmt.Errorf(format, data...))
}
func NewError(offset int, err error) *ErrorWithStack {
	if err == nil {
		panic("NewError called with nil error")
	}
	return &ErrorWithStack{
		offset: offset,
		Err:    err,
		stack:  debug.Stack(),
	}
}

type Panic struct {
	Err        any
	Stacktrace []StackLine
	Stack      []byte
	PC         []uintptr
}

func (p Panic) Unwrap() error {
	err, ok := p.Err.(error)
	if !ok {
		return nil
	}
	return err
}

func (p Panic) Error() string {
	if err, ok := p.Err.(error); ok {
		return "panic: " + err.Error()
	}
	return "panic: " + fmt.Sprint(p.Err)
}
func (p Panic) String() string {
	s := fmt.Sprintf("panic: %s\n", p.Err)
	for _, sl := range p.Stacktrace {
		s += sl.String() + "\n"
	}
	return s
}

var path = ""

func init() {
	path, _ = os.Getwd()
}

// RelFile returns the relative path of the file
func (sl StackLine) RelFile() string {

	f2, err := filepath.Rel(path, sl.File)
	if err != nil {
		return sl.File
	}

	if len(f2) < len(sl.File) {
		return f2
	}
	return sl.File
}

func (sl StackLine) String() string {
	return fmt.Sprintf("%3d: %40q %45q\t%s:%s\t", sl.L, sl.Package, sl.Func, sl.RelFile(), sl.Line)
}

// NewPanic will generate a new panic that can be used as a normal error
// It is meant to be used with the recover() function
//
//	if err := recover() ; err != nil {
//	    err := NewErrPanic(err, debug.Stack())
//	    ...
//	}
func NewPanic(err any, data []byte, skip int) (p Panic) {
	p.Err = err
	p.Stacktrace = StackDecode(data)
	p.PC = make([]uintptr, 100)

	runtime.Callers(skip+2, p.PC)

	return
}

func (p Panic) StackFrames() []uintptr {
	return p.PC
}
