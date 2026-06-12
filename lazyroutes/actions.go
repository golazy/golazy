package lazyroutes

import (
	"context"
	"errors"
	"net/http"

	"golazy.dev/lazycontroller"
)

type Action func(http.ResponseWriter, *http.Request) error

type Factory[T any] func(context.Context) (*T, error)

func Bind[T any](
	ctx context.Context,
	factory Factory[T],
	action func(*T, http.ResponseWriter, *http.Request) error,
) http.Handler {
	return Handle(func(w http.ResponseWriter, r *http.Request) error {
		controller, err := factory(lazycontroller.WithWriter(ctx, w))
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
			var httpError *lazycontroller.HTTPError
			if errors.As(err, &httpError) {
				status = httpError.Status
			}
			http.Error(w, http.StatusText(status), status)
		}
	})
}
