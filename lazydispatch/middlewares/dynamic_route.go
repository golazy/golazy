package middlewares

import (
	"context"
	"net/http"
	"strings"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazytelemetry/lazytracing"
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
		_ = runDynamicRouteRegion(r.Context(), "dynamic_route.method_override", func() error {
			r, ok = middleware.applyMethodOverride(w, r)
			return nil
		})
		if !ok {
			return
		}

		buffer := lazydispatch.AcquireBufferedResponseWriter(w)
		defer lazydispatch.ReleaseBufferedResponseWriter(buffer)
		_ = runDynamicRouteRequestRegion(r.Context(), "dynamic_route.downstream", r, func(request *http.Request) error {
			errorHandler.ServeHTTP(buffer, request)
			return nil
		})
		_ = runDynamicRouteRegion(r.Context(), "dynamic_route.etag", func() error {
			lazydispatch.ApplyETag(buffer, r)
			return nil
		})
		_ = runDynamicRouteRegion(r.Context(), "dynamic_route.flush", func() error {
			return buffer.Flush()
		})
	})
}

func runDynamicRouteRegion(ctx context.Context, name string, fn func() error) error {
	_, span := startDynamicRouteRegion(ctx, name)
	if span == nil {
		return fn()
	}
	defer span.End()
	if err := fn(); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func runDynamicRouteRequestRegion(ctx context.Context, name string, r *http.Request, fn func(*http.Request) error) error {
	regionCtx, span := startDynamicRouteRegion(ctx, name)
	if span == nil {
		return fn(r)
	}
	defer span.End()
	if err := fn(r.WithContext(regionCtx)); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func startDynamicRouteRegion(ctx context.Context, name string) (context.Context, *lazytracing.Span) {
	if lazytracing.SpanFromContext(ctx) == nil {
		return ctx, nil
	}
	return lazytracing.StartRegion(ctx, name)
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
