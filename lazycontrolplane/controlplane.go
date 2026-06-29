package lazycontrolplane

import (
	"context"
	"fmt"
	"html"
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
	mux       *http.ServeMux
	paths     map[string]struct{}
	prefixes  []string
	endpoints []endpoint
	checks    []ReadinessCheck
	pprof     bool
}

type endpoint struct {
	Method string
	Path   string
	Prefix bool
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
		plane.EnablePprof()
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
	plane.addEndpoint(endpointFromPattern(pattern, path))
}

// Handle registers an exact control-plane endpoint.
func (plane *ControlPlane) Handle(pattern string, handler http.Handler) {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	plane.handle(pattern, controlPlanePatternPath(pattern), handler)
}

// AddReadinessCheck appends a readiness check to /readyz.
func (plane *ControlPlane) AddReadinessCheck(check ReadinessCheck) {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	if check.Check == nil {
		panic("lazycontrolplane: readiness check is nil")
	}
	plane.checks = append(plane.checks, check)
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

// EnablePprof registers the standard net/http/pprof handlers.
//
// It is safe to call EnablePprof more than once.
func (plane *ControlPlane) EnablePprof() {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	if plane.pprof {
		return
	}
	plane.mux.HandleFunc("/debug/pprof/", pprof.Index)
	plane.mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	plane.mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	plane.mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	plane.mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	plane.paths["/debug/pprof"] = struct{}{}
	plane.prefixes = append(plane.prefixes, "/debug/pprof/")
	for _, path := range []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/profile",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
	} {
		plane.addEndpoint(endpoint{Method: http.MethodGet, Path: path, Prefix: path == "/debug/pprof/"})
	}
	plane.pprof = true
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

// StandaloneHandler serves the control plane on its own listener.
//
// It adds a root HTML index that lists registered endpoints. Use Handler when
// the control plane shares an application listener so "/" stays owned by the
// app.
func (plane *ControlPlane) StandaloneHandler() http.Handler {
	if plane == nil {
		return http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			plane.serveIndex(w, r)
			return
		}
		plane.ServeHTTP(w, r)
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

func endpointFromPattern(pattern, path string) endpoint {
	fields := strings.Fields(pattern)
	method := "ANY"
	if len(fields) > 1 {
		method = strings.ToUpper(fields[0])
	}
	return endpoint{Method: method, Path: path}
}

func (plane *ControlPlane) addEndpoint(next endpoint) {
	if plane == nil || next.Path == "" {
		return
	}
	for _, existing := range plane.endpoints {
		if existing.Method == next.Method && existing.Path == next.Path && existing.Prefix == next.Prefix {
			return
		}
	}
	plane.endpoints = append(plane.endpoints, next)
}

func (plane *ControlPlane) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}

	endpoints := append([]endpoint(nil), plane.endpoints...)
	var body strings.Builder
	body.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">`)
	body.WriteString(`<title>GoLazy Control Plane</title>`)
	body.WriteString(`<style>`)
	body.WriteString(`:root{color-scheme:light dark;--bg:#f6f7f9;--panel:#fff;--panel2:#eef3f8;--text:#17202a;--muted:#647083;--border:#d7dee8;--accent:#2563eb;--ok:#15803d;--warn:#a16207;--code:#0f172a}@media(prefers-color-scheme:dark){:root{--bg:#111418;--panel:#1a1f25;--panel2:#202832;--text:#e5e7eb;--muted:#9aa4b2;--border:#303a46;--accent:#60a5fa;--ok:#4ade80;--warn:#facc15;--code:#e5e7eb}}*{box-sizing:border-box}body{background:var(--bg);color:var(--text);font:14px/1.45 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:0}.shell{margin:0 auto;max-width:1080px;padding:28px 18px 40px}.hero{align-items:end;display:grid;gap:14px;grid-template-columns:minmax(0,1fr) auto;margin-bottom:18px}.eyebrow{color:var(--muted);font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}h1{font-size:28px;line-height:1.1;margin:4px 0 0}.summary{display:grid;gap:10px;grid-template-columns:repeat(3,minmax(0,1fr));margin:18px 0}.metric{background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:12px}.metric span{color:var(--muted);display:block;font-size:12px}.metric strong{display:block;font-size:24px;line-height:1.1;margin-top:3px}.card{background:var(--panel);border:1px solid var(--border);border-radius:8px;overflow:hidden}.toolbar{align-items:center;background:var(--panel2);border-bottom:1px solid var(--border);display:flex;gap:10px;justify-content:space-between;padding:10px 12px}.toolbar strong{font-size:13px}.toolbar span{color:var(--muted);font-size:12px}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid var(--border);padding:9px 12px;text-align:left;white-space:nowrap}th{color:var(--muted);font-size:12px;font-weight:700}td code{color:var(--code);font-family:ui-monospace,SFMono-Regular,Consolas,monospace}.method{border-radius:999px;display:inline-block;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12px;font-weight:700;min-width:48px;padding:2px 7px;text-align:center}.method-get{background:color-mix(in srgb,var(--ok) 16%,transparent);color:var(--ok)}.method-post{background:color-mix(in srgb,var(--accent) 16%,transparent);color:var(--accent)}.method-any{background:color-mix(in srgb,var(--warn) 18%,transparent);color:var(--warn)}.muted{color:var(--muted)}@media(max-width:760px){.hero{display:block}.summary{grid-template-columns:1fr}th,td{padding:8px}.shell{padding:20px 12px 30px}}`)
	body.WriteString(`</style></head><body><main class="shell">`)
	body.WriteString(`<section class="hero"><div><div class="eyebrow">GoLazy Operations</div><h1>Control Plane</h1></div><div class="muted">Registered endpoints</div></section>`)
	fmt.Fprintf(&body, `<section class="summary"><div class="metric"><span>Total endpoints</span><strong>%d</strong></div><div class="metric"><span>Health probes</span><strong>%d</strong></div><div class="metric"><span>Diagnostics</span><strong>%d</strong></div></section>`, len(endpoints), countEndpointPrefix(endpoints, "/livez", "/readyz"), countEndpointPrefix(endpoints, "/debug/pprof"))
	body.WriteString(`<section class="card"><div class="toolbar"><strong>Endpoints</strong><span>Served from this control-plane listener</span></div><div class="table-wrap"><table><thead><tr><th>Method</th><th>Path</th><th>Type</th></tr></thead><tbody>`)
	for _, endpoint := range endpoints {
		methodClass := "method-any"
		switch endpoint.Method {
		case http.MethodGet:
			methodClass = "method-get"
		case http.MethodPost:
			methodClass = "method-post"
		}
		kind := "Exact"
		if endpoint.Prefix {
			kind = "Prefix"
		}
		fmt.Fprintf(&body, `<tr><td><span class="method %s">%s</span></td><td><code>%s</code></td><td class="muted">%s</td></tr>`, methodClass, html.EscapeString(endpoint.Method), html.EscapeString(endpoint.Path), kind)
	}
	if len(endpoints) == 0 {
		body.WriteString(`<tr><td colspan="3" class="muted">No endpoints registered.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div></section></main></body></html>`)
	_, _ = w.Write([]byte(body.String()))
}

func countEndpointPrefix(endpoints []endpoint, prefixes ...string) int {
	count := 0
	for _, endpoint := range endpoints {
		for _, prefix := range prefixes {
			if endpoint.Path == prefix || strings.HasPrefix(endpoint.Path, prefix) {
				count++
				break
			}
		}
	}
	return count
}
