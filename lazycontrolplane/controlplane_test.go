package lazycontrolplane

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var _ Builder = Config{}
var _ Builder = (*ControlPlane)(nil)

func TestEmptyConfigServesLiveAndReady(t *testing.T) {
	plane := New(Config{})

	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/livez", body: "live\n"},
		{path: "/readyz", body: "ready\n"},
	} {
		response := httptest.NewRecorder()
		plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, http.StatusOK)
		}
		if got := response.Body.String(); got != test.body {
			t.Fatalf("%s body = %q, want %q", test.path, got, test.body)
		}
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s Cache-Control = %q, want no-store", test.path, got)
		}
	}
}

func TestReadyzReportsFailedCheck(t *testing.T) {
	plane := New(Config{
		Readiness: []ReadinessCheck{{
			Name: "database",
			Check: func(context.Context) error {
				return errors.New("connection refused")
			},
		}},
	})

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if got := response.Body.String(); !strings.Contains(got, "not ready: database: connection refused") {
		t.Fatalf("body = %q, want failed check", got)
	}
}

func TestMetricsIsOptional(t *testing.T) {
	plane := New(Config{})

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}

	plane = New(Config{
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "sample_metric 1\n")
		}),
	})
	response = httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("configured status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got != "sample_metric 1\n" {
		t.Fatalf("configured body = %q, want metric", got)
	}
}

func TestHandlerMountsControlPlaneBeforeNext(t *testing.T) {
	plane := New(Config{})
	handler := plane.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "app")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "live\n" {
		t.Fatalf("/livez body = %q, want control plane", got)
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app", nil))
	if got := response.Body.String(); got != "app" {
		t.Fatalf("/app body = %q, want next handler", got)
	}
}

func TestPprofIsExplicit(t *testing.T) {
	plane := New(Config{})
	if plane.HandlesPath("/debug/pprof/") {
		t.Fatal("pprof path is handled by default")
	}

	plane = New(Config{Pprof: true})
	if !plane.HandlesPath("/debug/pprof/") {
		t.Fatal("pprof path is not handled when enabled")
	}
	if !plane.HandlesPath("/debug/pprof/profile") {
		t.Fatal("pprof profile path is not handled when enabled")
	}
}
