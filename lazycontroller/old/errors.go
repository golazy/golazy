package lazycontroller

import (
	"fmt"
	"net/http"
	"strings"

	"golazy.dev/lazysupport"
)

func errString(err error) string {
	if errs, ok := err.(fmt.Stringer); ok {
		return errs.String()
	}
	return err.Error()
}

func shortError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if i := strings.IndexRune(msg, '\n'); i != -1 {
		msg = msg[:i]
	}
	return msg

}
func cleanError(err error) string {
	msg := strings.TrimSpace(err.Error())
	if i := strings.IndexRune(msg, '\n'); i != -1 {
		msg = msg[:i]
	}
	if err, ok := err.(*lazysupport.ErrorWithStack); ok {
		s := err.Stacktrace()
		if len(s.StackLines) > 0 {
			msg += " " + s.StackLines[0].RelFile() + ":" + s.StackLines[0].Line + " " + s.StackLines[0].Func
		}
	}
	return msg
}

type ErrorWithHandler interface {
	Handler() http.Handler
}

// Error represents a generic error
type Error struct {
	error

	// Location is the destination of the error
	// If set, lazydispatch will:
	// * Set the Location header
	// * Default the http status code to 302
	// * Set error.Error() as a flash message
	Location string

	// Code is the http status code to return
	Code int

	// Raise is a flag to raise the error as exception after sending the resposne to the client
	Raise bool

	// Public is a flag to indicate if the error is public
	// If set, the user will see the details of the error message
	// Otherwise, the user will see a generic error message
	// NOTE: This only applies to environments that are not development or tests
	Public bool

	// NoFlash ensure that a flash is not generated to the user
	NoFlash bool

	ErrTitle   string
	ErrMessage string
}

func (e Error) Unwrap() error {
	return e.error
}
func (e Error) Title() string {
	if e.ErrTitle != "" {
		return e.ErrTitle
	}
	if e.ErrMessage != "" {
		return lazysupport.Humanize(e.error.Error())
	}
	return ""
}

func (e Error) Message() string {
	if e.ErrMessage != "" {
		return e.ErrMessage
	}
	if e.ErrTitle != "" {
		return ""
	}
	return e.error.Error()
}

func (e Error) String() string {
	if err, ok := e.error.(fmt.Stringer); ok {
		return err.String()
	}
	return e.error.Error()
}

type ErrOpts struct {
	Err        error
	Message    string
	Title      string
	Location   string
	Code       int
	Offset     int
	Stacktrace bool
	Format     string
	FormatArgs []any
}

func NewError(opts ErrOpts) *Error {
	err := &Error{
		error:      opts.Err,
		ErrTitle:   opts.Title,
		ErrMessage: opts.Message,
		Location:   opts.Location,
		Code:       opts.Code,
	}
	if opts.Err == nil && opts.Format != "" {
		err.error = fmt.Errorf(opts.Format, opts.FormatArgs...)
	}
	if opts.Stacktrace {
		err.error = lazysupport.NewError(opts.Offset+1, err.error)
	}
	return err
}

//	func NewError(offset int, err error) *Error {
//		return &Error{
//			error: lazysupport.NewError(offset+1, err),
//		}
//	}
func NewRedirectErrorf(offset int, location string, format string, data ...any) *Error {
	err := NewErrorf(offset+1, format, data...)
	err.Location = location
	return err
}

func NewRedirectError(offset int, location string, err error) *Error {
	e := NewError(ErrOpts{Offset: offset + 1, Err: err})
	e.Location = location
	return e
}

type RedirectError struct {
	Location string
	Code     int
}

func (e *RedirectError) Error() string {
	return "redirecting..."
}

func (e *RedirectError) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e.Code == 0 {
		e.Code = 303
	}
	http.Redirect(w, r, e.Location, e.Code)

}

func NewErrorf(offset int, format string, data ...any) *Error {
	err := lazysupport.NewErrorf(offset+1, format, data...)
	return &Error{
		error: err,
	}

}

var ErrNotFound = &Error{Code: 404}
var ErrForbidden = &Error{Code: 403}
var ErrBadRequest = &Error{Code: 400}
var ErrUnauthorized = &Error{Code: 401}
var ErrUnprocessableEntity = &Error{Code: 422}
