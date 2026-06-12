package routes

import (
	"context"
	"fmt"
	"net/http"
)

type publicContextKey struct{}

func WithPublic(ctx context.Context, public http.Handler) context.Context {
	return context.WithValue(ctx, publicContextKey{}, public)
}

func New(ctx context.Context) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/", publicFallback(ctx))
	return mux
}

func MethodNotAllowed(allowed ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for _, method := range allowed {
			w.Header().Add("Allow", method)
		}
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	})
}

func publicFallback(ctx context.Context) http.Handler {
	public, ok := ctx.Value(publicContextKey{}).(http.Handler)
	if !ok {
		panic(fmt.Errorf("public file handler is missing from application context"))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			MethodNotAllowed(http.MethodGet).ServeHTTP(w, r)
			return
		}
		public.ServeHTTP(w, r)
	})
}
