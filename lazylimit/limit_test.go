package lazylimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMemoryLimiterFixedWindow(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter := NewMemory(2, time.Minute)
	limiter.Now = func() time.Time { return now }
	for i := 0; i < 2; i++ {
		result, err := limiter.Allow(context.Background(), "alice")
		if err != nil {
			t.Fatal(err)
		}
		if !result.Allowed {
			t.Fatalf("attempt %d denied", i+1)
		}
	}
	result, err := limiter.Allow(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed {
		t.Fatal("third attempt allowed")
	}
	now = now.Add(time.Minute)
	result, err = limiter.Allow(context.Background(), "alice")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatal("attempt after reset denied")
	}
}

func TestMiddlewareRejectsLimitedRequest(t *testing.T) {
	limiter := NewMemory(1, time.Minute)
	handler := Middleware(limiter, func(*http.Request) string { return "key" })(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusTooManyRequests)
	}
}
