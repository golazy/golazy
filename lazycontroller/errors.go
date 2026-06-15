package lazycontroller

import (
	"errors"
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

func StatusCode(err error) int {
	status := http.StatusInternalServerError
	var httpError *HTTPError
	if errors.As(err, &httpError) && httpError.Status != 0 {
		status = httpError.Status
	}
	return status
}
