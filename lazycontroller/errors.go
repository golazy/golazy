package lazycontroller

import (
	"fmt"
	"net/http"
)

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
