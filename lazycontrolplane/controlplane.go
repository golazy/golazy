package lazycontrolplane

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
)

// Builder creates a control plane for lazyapp.
//
// Config and *ControlPlane implement Builder. This keeps lazyapp.Config's
// ControlPlane field optional while still allowing ControlPlane: Config{}.
type Builder interface {
	BuildControlPlane() *ControlPlane
}

// Config describes the operational endpoints exposed by a control plane.
//
// The zero Config exposes /livez and /readyz.
type Config struct {
	Readiness []ReadinessCheck
	Metrics   http.Handler
	Pprof     bool
}

// ReadinessCheck is evaluated by /readyz.
type ReadinessCheck struct {
	Name  string
	Check func(context.Context) error
}

// ControlPlane routes operational endpoints.
type ControlPlane struct {
	mux      *http.ServeMux
	paths    map[string]struct{}
	prefixes []string
	checks   []ReadinessCheck
}

// New builds a control plane from config.
func New(config Config) *ControlPlane {
	for _, check := range config.Readiness {
		if check.Check == nil {
			panic("lazycontrolplane: readiness check is nil")
		}
	}

	plane := &ControlPlane{
		mux:    http.NewServeMux(),
		paths:  map[string]struct{}{},
		checks: append([]ReadinessCheck(nil), config.Readiness...),
	}
	plane.handle("GET /livez", "/livez", http.HandlerFunc(plane.livez))
	plane.handle("GET /readyz", "/readyz", http.HandlerFunc(plane.readyz))
	if config.Metrics != nil {
		plane.handle("GET /metrics", "/metrics", config.Metrics)
	}
	if config.Pprof {
		plane.mountPprof()
	}
	return plane
}

// BuildControlPlane implements Builder.
func (config Config) BuildControlPlane() *ControlPlane {
	return New(config)
}

// BuildControlPlane implements Builder.
func (plane *ControlPlane) BuildControlPlane() *ControlPlane {
	return plane
}

func (plane *ControlPlane) handle(pattern, path string, handler http.Handler) {
	plane.mux.Handle(pattern, handler)
	plane.paths[path] = struct{}{}
}

// Handle registers an exact control-plane endpoint.
func (plane *ControlPlane) Handle(pattern string, handler http.Handler) {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	plane.handle(pattern, controlPlanePatternPath(pattern), handler)
}

func controlPlanePatternPath(pattern string) string {
	fields := strings.Fields(pattern)
	if len(fields) == 0 {
		panic("lazycontrolplane: route pattern is empty")
	}
	path := fields[0]
	if len(fields) > 1 {
		path = fields[1]
	}
	if !strings.HasPrefix(path, "/") {
		panic("lazycontrolplane: route pattern path must start with /")
	}
	return path
}

func (plane *ControlPlane) mountPprof() {
	plane.mux.HandleFunc("/debug/pprof/", pprof.Index)
	plane.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	plane.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	plane.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	plane.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	plane.paths["/debug/pprof"] = struct{}{}
	plane.prefixes = append(plane.prefixes, "/debug/pprof/")
}

// HandlesPath reports whether path belongs to the control plane.
func (plane *ControlPlane) HandlesPath(path string) bool {
	if plane == nil {
		return false
	}
	if _, ok := plane.paths[path]; ok {
		return true
	}
	for _, prefix := range plane.prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// Handler mounts the control plane in front of next.
func (plane *ControlPlane) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if plane != nil && plane.HandlesPath(r.URL.Path) {
			plane.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ServeHTTP serves control-plane endpoints.
func (plane *ControlPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if plane == nil {
		http.NotFound(w, r)
		return
	}
	plane.mux.ServeHTTP(w, r)
}

func (plane *ControlPlane) livez(w http.ResponseWriter, _ *http.Request) {
	writePlain(w, http.StatusOK, "live\n")
}

func (plane *ControlPlane) readyz(w http.ResponseWriter, r *http.Request) {
	for _, check := range plane.checks {
		if err := check.Check(r.Context()); err != nil {
			name := check.Name
			if name == "" {
				name = "readiness"
			}
			writePlain(w, http.StatusServiceUnavailable, fmt.Sprintf("not ready: %s: %v\n", name, err))
			return
		}
	}
	writePlain(w, http.StatusOK, "ready\n")
}

func writePlain(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
