package lazycontrolplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/http/pprof"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// ErrSealed is returned when code tries to extend a control plane that has
// already been prepared for serving.
var ErrSealed = errors.New("lazycontrolplane: registrations are sealed")

// PanelsPath is the control-plane discovery endpoint for developer-panel
// contributions registered by the running application and its add-ons.
const PanelsPath = "/_golazy/panels"

// Builder creates a control plane for lazyapp.
//
// Config and *ControlPlane implement Builder. This keeps lazyapp.Config's
// ControlPlane field optional while still allowing ControlPlane: Config{}.
type Builder interface {
	BuildControlPlane() *ControlPlane
}

// Config describes the operational endpoints exposed by a control plane.
//
// The zero Config exposes /livez, /readyz, and panel discovery.
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

// Endpoint describes an owned control-plane endpoint.
//
// ID must be unique within a control plane. Owner identifies the framework
// package, application, or add-on responsible for the endpoint. Pattern uses
// net/http ServeMux syntax.
type Endpoint struct {
	ID          string
	Owner       string
	Pattern     string
	Description string
	Handler     http.Handler
}

// EndpointInfo is the public, handler-free description of a registered
// endpoint.
type EndpointInfo struct {
	ID          string
	Owner       string
	Pattern     string
	Description string
	Method      string
	Path        string
	Prefix      bool
}

// Panel describes one developer-panel entry backed by an owned control-plane
// endpoint. EndpointID must name a GET-capable endpoint registered by the same
// Owner. Actions may reference exact POST-only endpoints owned by that Owner.
// The endpoint handlers continue to run in the application process; panel
// hosts consume only the metadata returned by PanelsPath.
type Panel struct {
	ID          string
	Owner       string
	Title       string
	Description string
	EndpointID  string
	Actions     []PanelAction
	Order       int
}

// PanelAction describes a trusted action rendered by a developer-panel host.
// EndpointID must name an exact POST endpoint registered by the panel Owner.
// Add-on content cannot submit this endpoint directly; the host resolves the
// stable panel and action IDs from discovery before issuing an empty POST.
type PanelAction struct {
	ID          string
	Title       string
	Description string
	EndpointID  string
}

// PanelInfo is the resolved, handler-free description exposed to developer
// panel hosts.
type PanelInfo struct {
	ID          string            `json:"id"`
	Owner       string            `json:"owner"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	EndpointID  string            `json:"endpoint_id"`
	Pattern     string            `json:"pattern"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Actions     []PanelActionInfo `json:"actions,omitempty"`
	Order       int               `json:"order"`
}

// PanelActionInfo is the resolved, handler-free action metadata exposed to
// developer-panel hosts.
type PanelActionInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	EndpointID  string `json:"endpoint_id"`
	Pattern     string `json:"pattern"`
	Method      string `json:"method"`
	Path        string `json:"path"`
}

// PanelsResponse is the versioned response served from PanelsPath.
type PanelsResponse struct {
	Schema int         `json:"schema"`
	Panels []PanelInfo `json:"panels"`
}

// Registrar accepts owned endpoints and readiness checks during control-plane
// setup. Applications close registration explicitly with ControlPlane.Seal
// after startup packages and add-ons finish registering.
type Registrar interface {
	Register(Endpoint) error
	RegisterReadinessCheck(ReadinessCheck) error
	RegisterPanel(Panel) error
}

// ControlPlane routes operational endpoints.
type ControlPlane struct {
	mu        sync.RWMutex
	mux       *http.ServeMux
	paths     map[string]struct{}
	prefixes  []string
	endpoints []EndpointInfo
	ids       map[string]struct{}
	patterns  map[string]struct{}
	byID      map[string]EndpointInfo
	panels    []PanelInfo
	panelIDs  map[string]struct{}
	checks    []ReadinessCheck
	pprof     bool
	sealed    bool
}

var _ Registrar = (*ControlPlane)(nil)

// New builds a control plane from config.
func New(config Config) *ControlPlane {
	for _, check := range config.Readiness {
		if check.Check == nil {
			panic("lazycontrolplane: readiness check is nil")
		}
	}

	plane := &ControlPlane{
		mux:      http.NewServeMux(),
		paths:    map[string]struct{}{},
		ids:      map[string]struct{}{},
		patterns: map[string]struct{}{},
		byID:     map[string]EndpointInfo{},
		panelIDs: map[string]struct{}{},
		checks:   append([]ReadinessCheck(nil), config.Readiness...),
	}
	plane.mustRegister(Endpoint{
		ID:          "lazycontrolplane.livez",
		Owner:       "golazy.dev/lazycontrolplane",
		Pattern:     "GET /livez",
		Description: "Process liveness",
		Handler:     http.HandlerFunc(plane.livez),
	})
	plane.mustRegister(Endpoint{
		ID:          "lazycontrolplane.readyz",
		Owner:       "golazy.dev/lazycontrolplane",
		Pattern:     "GET /readyz",
		Description: "Application readiness",
		Handler:     http.HandlerFunc(plane.readyz),
	})
	plane.mustRegister(Endpoint{
		ID:          "lazycontrolplane.panels",
		Owner:       "golazy.dev/lazycontrolplane",
		Pattern:     "GET " + PanelsPath,
		Description: "Developer-panel contribution discovery",
		Handler:     http.HandlerFunc(plane.servePanels),
	})
	if config.Metrics != nil {
		plane.mustRegister(Endpoint{
			ID:          "lazycontrolplane.metrics",
			Owner:       "golazy.dev/lazycontrolplane",
			Pattern:     "GET /metrics",
			Description: "Application metrics",
			Handler:     config.Metrics,
		})
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

// Register adds an owned endpoint to the control plane.
//
// Registration is concurrency-safe. It fails when ID or Pattern is already
// registered, when the endpoint is invalid, or after the control plane has
// been sealed for serving.
func (plane *ControlPlane) Register(endpoint Endpoint) error {
	if plane == nil {
		return errors.New("lazycontrolplane: control plane is nil")
	}
	plane.mu.Lock()
	defer plane.mu.Unlock()
	return plane.registerLocked(endpoint)
}

func (plane *ControlPlane) mustRegister(endpoint Endpoint) {
	if err := plane.Register(endpoint); err != nil {
		panic(err)
	}
}

// Handle registers an exact control-plane endpoint.
//
// Handle is the compatibility API for registrations without ownership
// metadata. New integrations should use Register so validation errors can be
// handled by the caller.
func (plane *ControlPlane) Handle(pattern string, handler http.Handler) {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	plane.mustRegister(Endpoint{
		ID:          "legacy:" + pattern,
		Owner:       "legacy",
		Pattern:     pattern,
		Description: "Registered through ControlPlane.Handle",
		Handler:     handler,
	})
}

// AddReadinessCheck appends a readiness check to /readyz.
//
// AddReadinessCheck is the compatibility API. New integrations should use
// RegisterReadinessCheck so validation errors can be handled by the caller.
func (plane *ControlPlane) AddReadinessCheck(check ReadinessCheck) {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	if err := plane.RegisterReadinessCheck(check); err != nil {
		panic(err)
	}
}

// RegisterReadinessCheck appends a readiness check to /readyz.
func (plane *ControlPlane) RegisterReadinessCheck(check ReadinessCheck) error {
	if plane == nil {
		return errors.New("lazycontrolplane: control plane is nil")
	}
	if check.Check == nil {
		return errors.New("lazycontrolplane: readiness check is nil")
	}
	plane.mu.Lock()
	defer plane.mu.Unlock()
	if plane.sealed {
		return ErrSealed
	}
	plane.checks = append(plane.checks, check)
	return nil
}

// RegisterPanel adds a developer-panel entry backed by an endpoint registered
// on this control plane. The endpoint must belong to the same owner, support
// GET requests, and use an exact path. Prefix and wildcard patterns are
// rejected so a panel host cannot proxy into another endpoint's namespace.
func (plane *ControlPlane) RegisterPanel(panel Panel) error {
	if plane == nil {
		return errors.New("lazycontrolplane: control plane is nil")
	}
	plane.mu.Lock()
	defer plane.mu.Unlock()
	if plane.sealed {
		return ErrSealed
	}

	panel.ID = strings.TrimSpace(panel.ID)
	if panel.ID == "" {
		return errors.New("lazycontrolplane: panel ID is empty")
	}
	panel.Owner = strings.TrimSpace(panel.Owner)
	if panel.Owner == "" {
		return fmt.Errorf("lazycontrolplane: panel %q owner is empty", panel.ID)
	}
	panel.Title = strings.TrimSpace(panel.Title)
	if panel.Title == "" {
		return fmt.Errorf("lazycontrolplane: panel %q title is empty", panel.ID)
	}
	panel.Description = strings.TrimSpace(panel.Description)
	if panel.Description == "" {
		return fmt.Errorf("lazycontrolplane: panel %q description is empty", panel.ID)
	}
	panel.EndpointID = strings.TrimSpace(panel.EndpointID)
	if panel.EndpointID == "" {
		return fmt.Errorf("lazycontrolplane: panel %q endpoint ID is empty", panel.ID)
	}
	if _, exists := plane.panelIDs[panel.ID]; exists {
		return fmt.Errorf("lazycontrolplane: panel ID %q is already registered", panel.ID)
	}
	endpoint, exists := plane.byID[panel.EndpointID]
	if !exists {
		return fmt.Errorf("lazycontrolplane: panel %q endpoint %q is not registered", panel.ID, panel.EndpointID)
	}
	if endpoint.Owner != panel.Owner {
		return fmt.Errorf("lazycontrolplane: panel %q owner %q does not own endpoint %q", panel.ID, panel.Owner, panel.EndpointID)
	}
	if endpoint.Method != http.MethodGet && endpoint.Method != "ANY" {
		return fmt.Errorf("lazycontrolplane: panel %q endpoint %q does not support GET", panel.ID, panel.EndpointID)
	}
	if endpoint.Prefix {
		return fmt.Errorf("lazycontrolplane: panel %q endpoint %q uses a prefix pattern", panel.ID, panel.EndpointID)
	}
	if strings.Contains(endpoint.Path, "{") {
		return fmt.Errorf("lazycontrolplane: panel %q endpoint %q uses a wildcard pattern", panel.ID, panel.EndpointID)
	}

	actions, err := plane.resolvePanelActionsLocked(panel)
	if err != nil {
		return err
	}

	plane.panelIDs[panel.ID] = struct{}{}
	plane.panels = append(plane.panels, PanelInfo{
		ID:          panel.ID,
		Owner:       panel.Owner,
		Title:       panel.Title,
		Description: panel.Description,
		EndpointID:  endpoint.ID,
		Pattern:     endpoint.Pattern,
		Method:      endpoint.Method,
		Path:        endpoint.Path,
		Actions:     actions,
		Order:       panel.Order,
	})
	return nil
}

func (plane *ControlPlane) resolvePanelActionsLocked(panel Panel) ([]PanelActionInfo, error) {
	if len(panel.Actions) == 0 {
		return nil, nil
	}
	actions := make([]PanelActionInfo, 0, len(panel.Actions))
	ids := make(map[string]struct{}, len(panel.Actions))
	patterns := make(map[string]struct{}, len(panel.Actions))
	for index, action := range panel.Actions {
		action.ID = strings.TrimSpace(action.ID)
		if action.ID == "" {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %d ID is empty", panel.ID, index)
		}
		if _, exists := ids[action.ID]; exists {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action ID %q is duplicated", panel.ID, action.ID)
		}
		action.Title = strings.TrimSpace(action.Title)
		if action.Title == "" {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q title is empty", panel.ID, action.ID)
		}
		action.Description = strings.TrimSpace(action.Description)
		if action.Description == "" {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q description is empty", panel.ID, action.ID)
		}
		action.EndpointID = strings.TrimSpace(action.EndpointID)
		if action.EndpointID == "" {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q endpoint ID is empty", panel.ID, action.ID)
		}
		endpoint, exists := plane.byID[action.EndpointID]
		if !exists {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q endpoint %q is not registered", panel.ID, action.ID, action.EndpointID)
		}
		if endpoint.Owner != panel.Owner {
			return nil, fmt.Errorf("lazycontrolplane: panel %q owner %q does not own action %q endpoint %q", panel.ID, panel.Owner, action.ID, action.EndpointID)
		}
		if endpoint.Method != http.MethodPost {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q endpoint %q is not POST-only", panel.ID, action.ID, action.EndpointID)
		}
		if endpoint.Prefix {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q endpoint %q uses a prefix pattern", panel.ID, action.ID, action.EndpointID)
		}
		if strings.Contains(endpoint.Path, "{") {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q endpoint %q uses a wildcard pattern", panel.ID, action.ID, action.EndpointID)
		}
		if _, exists := patterns[endpoint.Pattern]; exists {
			return nil, fmt.Errorf("lazycontrolplane: panel %q action %q duplicates method and path %q", panel.ID, action.ID, endpoint.Pattern)
		}
		ids[action.ID] = struct{}{}
		patterns[endpoint.Pattern] = struct{}{}
		actions = append(actions, PanelActionInfo{
			ID:          action.ID,
			Title:       action.Title,
			Description: action.Description,
			EndpointID:  endpoint.ID,
			Pattern:     endpoint.Pattern,
			Method:      endpoint.Method,
			Path:        endpoint.Path,
		})
	}
	return actions, nil
}

// Seal closes endpoint and readiness registration. It is safe to call Seal
// more than once. The application lifecycle calls Seal once all package and
// add-on registrations are complete.
func (plane *ControlPlane) Seal() {
	if plane == nil {
		return
	}
	plane.mu.Lock()
	plane.sealed = true
	plane.mu.Unlock()
}

func (plane *ControlPlane) registerLocked(endpoint Endpoint) error {
	if plane.sealed {
		return ErrSealed
	}
	info, err := endpointInformation(endpoint)
	if err != nil {
		return err
	}
	if _, exists := plane.ids[info.ID]; exists {
		return fmt.Errorf("lazycontrolplane: endpoint ID %q is already registered", info.ID)
	}
	if _, exists := plane.patterns[info.Pattern]; exists {
		return fmt.Errorf("lazycontrolplane: endpoint pattern %q is already registered", info.Pattern)
	}
	if err := handleServeMux(plane.mux, info.Pattern, endpoint.Handler); err != nil {
		return fmt.Errorf("lazycontrolplane: register endpoint %q: %w", info.ID, err)
	}

	plane.ids[info.ID] = struct{}{}
	plane.patterns[info.Pattern] = struct{}{}
	plane.byID[info.ID] = info
	plane.paths[info.Path] = struct{}{}
	if info.Prefix {
		if withoutSlash := strings.TrimSuffix(info.Path, "/"); withoutSlash != "" {
			plane.paths[withoutSlash] = struct{}{}
		}
		plane.prefixes = append(plane.prefixes, info.Path)
	}
	plane.endpoints = append(plane.endpoints, info)
	return nil
}

func endpointInformation(endpoint Endpoint) (EndpointInfo, error) {
	id := strings.TrimSpace(endpoint.ID)
	if id == "" {
		return EndpointInfo{}, errors.New("lazycontrolplane: endpoint ID is empty")
	}
	owner := strings.TrimSpace(endpoint.Owner)
	if owner == "" {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q owner is empty", id)
	}
	description := strings.TrimSpace(endpoint.Description)
	if description == "" {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q description is empty", id)
	}
	if endpoint.Handler == nil {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q handler is nil", id)
	}

	pattern := strings.TrimSpace(endpoint.Pattern)
	fields := strings.Fields(pattern)
	if len(fields) == 0 {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q pattern is empty", id)
	}
	if len(fields) > 2 {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q pattern %q is invalid", id, pattern)
	}
	path := fields[0]
	method := "ANY"
	if len(fields) > 1 {
		method = strings.ToUpper(fields[0])
		path = fields[1]
	}
	if !strings.HasPrefix(path, "/") {
		return EndpointInfo{}, fmt.Errorf("lazycontrolplane: endpoint %q pattern path must start with /", id)
	}

	return EndpointInfo{
		ID:          id,
		Owner:       owner,
		Pattern:     pattern,
		Description: description,
		Method:      method,
		Path:        path,
		Prefix:      strings.HasSuffix(path, "/"),
	}, nil
}

func handleServeMux(mux *http.ServeMux, pattern string, handler http.Handler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("invalid or conflicting pattern %q: %v", pattern, recovered)
		}
	}()
	mux.Handle(pattern, handler)
	return nil
}

// EnablePprof registers the standard net/http/pprof handlers.
//
// It is safe to call EnablePprof more than once.
func (plane *ControlPlane) EnablePprof() {
	if plane == nil {
		panic("lazycontrolplane: control plane is nil")
	}
	plane.mu.Lock()
	defer plane.mu.Unlock()
	if plane.pprof {
		return
	}
	registrations := []Endpoint{
		{ID: "lazycontrolplane.pprof.index", Owner: "golazy.dev/lazycontrolplane", Pattern: "/debug/pprof/", Description: "Runtime profiling index", Handler: http.HandlerFunc(pprof.Index)},
		{ID: "lazycontrolplane.pprof.cmdline", Owner: "golazy.dev/lazycontrolplane", Pattern: "/debug/pprof/cmdline", Description: "Process command line", Handler: http.HandlerFunc(pprof.Cmdline)},
		{ID: "lazycontrolplane.pprof.profile", Owner: "golazy.dev/lazycontrolplane", Pattern: "/debug/pprof/profile", Description: "CPU profile", Handler: http.HandlerFunc(pprof.Profile)},
		{ID: "lazycontrolplane.pprof.symbol", Owner: "golazy.dev/lazycontrolplane", Pattern: "/debug/pprof/symbol", Description: "Program counter lookup", Handler: http.HandlerFunc(pprof.Symbol)},
		{ID: "lazycontrolplane.pprof.trace", Owner: "golazy.dev/lazycontrolplane", Pattern: "/debug/pprof/trace", Description: "Runtime execution trace", Handler: http.HandlerFunc(pprof.Trace)},
	}
	for _, registration := range registrations {
		if err := plane.registerLocked(registration); err != nil {
			panic(err)
		}
	}
	plane.pprof = true
}

// HandlesPath reports whether path belongs to the control plane.
func (plane *ControlPlane) HandlesPath(path string) bool {
	if plane == nil || !strings.HasPrefix(path, "/") {
		return false
	}
	plane.mu.RLock()
	if _, ok := plane.paths[path]; ok {
		plane.mu.RUnlock()
		return true
	}
	for _, prefix := range plane.prefixes {
		if strings.HasPrefix(path, prefix) {
			plane.mu.RUnlock()
			return true
		}
	}
	methods := make(map[string]struct{}, len(plane.endpoints))
	for _, endpoint := range plane.endpoints {
		method := endpoint.Method
		if method == "ANY" {
			method = http.MethodGet
		}
		methods[method] = struct{}{}
	}
	plane.mu.RUnlock()

	for method := range methods {
		request := &http.Request{Method: method, URL: &url.URL{Path: path}}
		if _, pattern := plane.mux.Handler(request); pattern != "" {
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
	plane.mu.RLock()
	checks := append([]ReadinessCheck(nil), plane.checks...)
	plane.mu.RUnlock()
	for _, check := range checks {
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

// Endpoints returns a deterministic snapshot of registered endpoint metadata.
func (plane *ControlPlane) Endpoints() []EndpointInfo {
	if plane == nil {
		return nil
	}
	plane.mu.RLock()
	endpoints := append([]EndpointInfo(nil), plane.endpoints...)
	plane.mu.RUnlock()
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].ID < endpoints[j].ID
	})
	return endpoints
}

// Panels returns a deterministic snapshot of registered developer-panel
// entries. Lower Order values sort first, followed by title and stable ID.
func (plane *ControlPlane) Panels() []PanelInfo {
	if plane == nil {
		return []PanelInfo{}
	}
	plane.mu.RLock()
	panels := append([]PanelInfo(nil), plane.panels...)
	for index := range panels {
		panels[index].Actions = append([]PanelActionInfo(nil), panels[index].Actions...)
	}
	plane.mu.RUnlock()
	sort.Slice(panels, func(i, j int) bool {
		if panels[i].Order != panels[j].Order {
			return panels[i].Order < panels[j].Order
		}
		if panels[i].Title != panels[j].Title {
			return panels[i].Title < panels[j].Title
		}
		return panels[i].ID < panels[j].ID
	})
	if panels == nil {
		return []PanelInfo{}
	}
	return panels
}

func (plane *ControlPlane) servePanels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if err := json.NewEncoder(w).Encode(PanelsResponse{Schema: 1, Panels: plane.Panels()}); err != nil {
		return
	}
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

	endpoints := plane.Endpoints()
	var body strings.Builder
	body.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">`)
	body.WriteString(`<title>GoLazy Control Plane</title>`)
	body.WriteString(`<style>`)
	body.WriteString(`:root{color-scheme:light dark;--bg:#f6f7f9;--panel:#fff;--panel2:#eef3f8;--text:#17202a;--muted:#647083;--border:#d7dee8;--accent:#2563eb;--ok:#15803d;--warn:#a16207;--code:#0f172a}@media(prefers-color-scheme:dark){:root{--bg:#111418;--panel:#1a1f25;--panel2:#202832;--text:#e5e7eb;--muted:#9aa4b2;--border:#303a46;--accent:#60a5fa;--ok:#4ade80;--warn:#facc15;--code:#e5e7eb}}*{box-sizing:border-box}body{background:var(--bg);color:var(--text);font:14px/1.45 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;margin:0}.shell{margin:0 auto;max-width:1080px;padding:28px 18px 40px}.hero{align-items:end;display:grid;gap:14px;grid-template-columns:minmax(0,1fr) auto;margin-bottom:18px}.eyebrow{color:var(--muted);font-size:12px;font-weight:700;letter-spacing:.08em;text-transform:uppercase}h1{font-size:28px;line-height:1.1;margin:4px 0 0}.summary{display:grid;gap:10px;grid-template-columns:repeat(3,minmax(0,1fr));margin:18px 0}.metric{background:var(--panel);border:1px solid var(--border);border-radius:8px;padding:12px}.metric span{color:var(--muted);display:block;font-size:12px}.metric strong{display:block;font-size:24px;line-height:1.1;margin-top:3px}.card{background:var(--panel);border:1px solid var(--border);border-radius:8px;overflow:hidden}.toolbar{align-items:center;background:var(--panel2);border-bottom:1px solid var(--border);display:flex;gap:10px;justify-content:space-between;padding:10px 12px}.toolbar strong{font-size:13px}.toolbar span{color:var(--muted);font-size:12px}.table-wrap{overflow:auto}table{border-collapse:collapse;width:100%}th,td{border-bottom:1px solid var(--border);padding:9px 12px;text-align:left;white-space:nowrap}th{color:var(--muted);font-size:12px;font-weight:700}td code{color:var(--code);font-family:ui-monospace,SFMono-Regular,Consolas,monospace}.method{border-radius:999px;display:inline-block;font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12px;font-weight:700;min-width:48px;padding:2px 7px;text-align:center}.method-get{background:color-mix(in srgb,var(--ok) 16%,transparent);color:var(--ok)}.method-post{background:color-mix(in srgb,var(--accent) 16%,transparent);color:var(--accent)}.method-any{background:color-mix(in srgb,var(--warn) 18%,transparent);color:var(--warn)}.muted{color:var(--muted)}@media(max-width:760px){.hero{display:block}.summary{grid-template-columns:1fr}th,td{padding:8px}.shell{padding:20px 12px 30px}}`)
	body.WriteString(`</style></head><body><main class="shell">`)
	body.WriteString(`<section class="hero"><div><div class="eyebrow">GoLazy Operations</div><h1>Control Plane</h1></div><div class="muted">Registered endpoints</div></section>`)
	fmt.Fprintf(&body, `<section class="summary"><div class="metric"><span>Total endpoints</span><strong>%d</strong></div><div class="metric"><span>Health probes</span><strong>%d</strong></div><div class="metric"><span>Diagnostics</span><strong>%d</strong></div></section>`, len(endpoints), countEndpointPrefix(endpoints, "/livez", "/readyz"), countEndpointPrefix(endpoints, "/debug/pprof"))
	body.WriteString(`<section class="card"><div class="toolbar"><strong>Endpoints</strong><span>Served from this control-plane listener</span></div><div class="table-wrap"><table><thead><tr><th>Method</th><th>Path</th><th>Owner</th><th>ID</th><th>Description</th><th>Type</th></tr></thead><tbody>`)
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
		} else if strings.Contains(endpoint.Path, "{") {
			kind = "Pattern"
		}
		fmt.Fprintf(&body, `<tr><td><span class="method %s">%s</span></td><td><code>%s</code></td><td><code>%s</code></td><td><code>%s</code></td><td class="muted">%s</td><td class="muted">%s</td></tr>`, methodClass, html.EscapeString(endpoint.Method), html.EscapeString(endpoint.Path), html.EscapeString(endpoint.Owner), html.EscapeString(endpoint.ID), html.EscapeString(endpoint.Description), kind)
	}
	if len(endpoints) == 0 {
		body.WriteString(`<tr><td colspan="6" class="muted">No endpoints registered.</td></tr>`)
	}
	body.WriteString(`</tbody></table></div></section></main></body></html>`)
	_, _ = w.Write([]byte(body.String()))
}

func countEndpointPrefix(endpoints []EndpointInfo, prefixes ...string) int {
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
