package lazylimit

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Result describes one rate-limit check.
type Result struct {
	Allowed    bool
	Remaining  int
	ResetAfter time.Duration
}

// Limiter checks whether key can consume one unit.
type Limiter interface {
	Allow(context.Context, string) (Result, error)
}

// MemoryLimiter is a fixed-window in-memory limiter.
type MemoryLimiter struct {
	Limit  int
	Window time.Duration
	Now    func() time.Time

	mu      sync.Mutex
	buckets map[string]bucket
}

type bucket struct {
	count int
	reset time.Time
}

// NewMemory creates an in-memory fixed-window limiter.
func NewMemory(limit int, window time.Duration) *MemoryLimiter {
	return &MemoryLimiter{Limit: limit, Window: window, buckets: map[string]bucket{}}
}

// Allow implements Limiter.
func (limiter *MemoryLimiter) Allow(_ context.Context, key string) (Result, error) {
	if limiter == nil || limiter.Limit <= 0 || limiter.Window <= 0 {
		return Result{Allowed: true, Remaining: -1}, nil
	}
	now := time.Now
	if limiter.Now != nil {
		now = limiter.Now
	}
	current := now()
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if limiter.buckets == nil {
		limiter.buckets = map[string]bucket{}
	}
	b := limiter.buckets[key]
	if b.reset.IsZero() || !current.Before(b.reset) {
		b = bucket{reset: current.Add(limiter.Window)}
	}
	if b.count >= limiter.Limit {
		limiter.buckets[key] = b
		return Result{Allowed: false, Remaining: 0, ResetAfter: time.Until(b.reset)}, nil
	}
	b.count++
	limiter.buckets[key] = b
	return Result{Allowed: true, Remaining: limiter.Limit - b.count, ResetAfter: time.Until(b.reset)}, nil
}

// Middleware installs limiter in a standard HTTP stack.
func Middleware(limiter Limiter, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			limitKey := r.RemoteAddr
			if key != nil {
				limitKey = key(r)
			}
			result, err := limiter.Allow(r.Context(), limitKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if !result.Allowed {
				w.Header().Set("Retry-After", retryAfter(result.ResetAfter))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func retryAfter(duration time.Duration) string {
	if duration <= 0 {
		return "0"
	}
	seconds := max(int(duration.Round(time.Second)/time.Second), 1)
	return strconv.Itoa(seconds)
}
