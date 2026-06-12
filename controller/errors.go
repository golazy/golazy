package controller

import (
	"context"
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

type Action func(http.ResponseWriter, *http.Request) error

type Factory[T any] func(context.Context) (*T, error)

func Bind[T any](
	ctx context.Context,
	factory Factory[T],
	action func(*T, http.ResponseWriter, *http.Request) error,
) http.Handler {
	return Handle(func(w http.ResponseWriter, r *http.Request) error {
		controller, err := factory(WithWriter(ctx, w))
		if err != nil {
			return err
		}
		return action(controller, w, r)
	})
}

func Handle(action Action) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := action(w, r); err != nil {
			status := http.StatusInternalServerError
			var httpError *HTTPError
			if errors.As(err, &httpError) {
				status = httpError.Status
			}
			http.Error(w, http.StatusText(status), status)
		}
	})
}
