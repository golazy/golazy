package middlewares

import (
	"context"
	"net/http"
	"strings"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazydispatch"
)

func DynamicRoute(ctx context.Context) lazydispatch.Middleware {
	return dynamicRoute{ctx: ctx, maxMethodOverrideScan: defaultMethodOverrideScan}
}

type dynamicRoute struct {
	ctx                   context.Context
	maxMethodOverrideScan int64
}

func (dynamicRoute) MiddlewareName() string {
	return "lazydispatch.middlewares.DynamicRoute"
}

func (middleware dynamicRoute) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	errorHandler := lazycontroller.ErrorHandler(middleware.ctx)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		r, ok = middleware.applyMethodOverride(w, r)
		if !ok {
			return
		}

		buffer := lazydispatch.AcquireBufferedResponseWriter(w)
		defer lazydispatch.ReleaseBufferedResponseWriter(buffer)
		errorHandler.ServeHTTP(buffer, r)
		lazydispatch.ApplyETag(buffer, r)
		_ = buffer.Flush()
	})
}

func (middleware dynamicRoute) applyMethodOverride(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	if shouldSkipMethodOverride(r) {
		return r, true
	}

	method, present, valid := readMethodOverride(r, middleware.maxMethodOverrideScan)
	if present && !valid {
		http.Error(w, "invalid _method", http.StatusBadRequest)
		return r, false
	}
	if valid {
		r = r.WithContext(context.WithValue(r.Context(), originalMethodKey{}, r.Method))
		r.Method = strings.ToUpper(method)
	}
	return r, true
}
