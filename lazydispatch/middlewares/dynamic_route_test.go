package middlewares

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazytelemetry/lazytracing"
)

func TestDynamicRouteAppliesETagAndNotModified(t *testing.T) {
	handler := DynamicRoute(context.Background()).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "hello")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	etag := testDynamicRouteETag("hello")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag = %q, want %q", got, etag)
	}
	if got := response.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want hello", got)
	}

	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotModified)
	}
	if got := response.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}

func TestDynamicRouteCreatesTraceRegions(t *testing.T) {
	ctx, root := lazytracing.StartSpan(context.Background(), "http.server.request")
	defer root.End()
	var handlerSpan *lazytracing.Span
	handler := DynamicRoute(context.Background()).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, handlerSpan = lazytracing.StartRegion(r.Context(), "handler.work")
		if handlerSpan != nil {
			handlerSpan.End()
		}
		_, _ = fmt.Fprint(w, "hello")
	}))

	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	names := dynamicRouteSpanNames(root)
	for _, want := range []string{
		"dynamic_route.method_override",
		"dynamic_route.downstream",
		"dynamic_route.etag",
		"dynamic_route.flush",
	} {
		if !containsDynamicRouteSpanName(names, want) {
			t.Fatalf("span names = %#v, want %q", names, want)
		}
	}
	downstream := findDynamicRouteSpan(root, "dynamic_route.downstream")
	if downstream == nil {
		t.Fatalf("span names = %#v, want dynamic_route.downstream", names)
	}
	if handlerSpan == nil {
		t.Fatalf("span names = %#v, want handler child span", names)
	}
	if handlerSpan.ParentID() != downstream.SpanID() {
		t.Fatalf("handler parent = %q, want downstream span %q", handlerSpan.ParentID(), downstream.SpanID())
	}
}

func TestDynamicRouteAppliesMethodOverrideAndReplaysBody(t *testing.T) {
	var gotMethod string
	var gotOriginal string
	var gotBody string
	handler := DynamicRoute(context.Background()).Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotOriginal = OriginalMethod(r)
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
	}))

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_method=patch&name=Ada"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotMethod != http.MethodPatch {
		t.Fatalf("method = %q, want PATCH", gotMethod)
	}
	if gotOriginal != http.MethodPost {
		t.Fatalf("original method = %q, want POST", gotOriginal)
	}
	if gotBody != "_method=patch&name=Ada" {
		t.Fatalf("body = %q, want replayed body", gotBody)
	}
}

func TestDynamicRouteRejectsInvalidMethodOverride(t *testing.T) {
	handler := DynamicRoute(context.Background()).Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called")
	}))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_method=trace"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestDynamicRouteHandlesControllerReportedErrors(t *testing.T) {
	ctx := lazycontroller.WithDetailErrors(context.Background())
	handler := DynamicRoute(ctx).Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "stale")
		if !lazycontroller.ReportError(r, nil, errors.New("boom")) {
			t.Fatal("controller error was not reported")
		}
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Body.String(); !strings.Contains(got, "boom") || strings.Contains(got, "stale") {
		t.Fatalf("body = %q, want detail error without stale body", got)
	}
	if got := response.Header().Get("ETag"); got != "" {
		t.Fatalf("ETag = %q, want empty", got)
	}
}

func testDynamicRouteETag(body string) string {
	sum := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%q", fmt.Sprintf("%x", sum[:]))
}

func dynamicRouteSpanNames(span *lazytracing.Span) []string {
	if span == nil {
		return nil
	}
	names := []string{span.Name()}
	for _, child := range span.Children() {
		names = append(names, dynamicRouteSpanNames(child)...)
	}
	return names
}

func containsDynamicRouteSpanName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

func findDynamicRouteSpan(span *lazytracing.Span, name string) *lazytracing.Span {
	if span == nil {
		return nil
	}
	if span.Name() == name {
		return span
	}
	for _, child := range span.Children() {
		if found := findDynamicRouteSpan(child, name); found != nil {
			return found
		}
	}
	return nil
}
