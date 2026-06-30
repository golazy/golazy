package lazyworkers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"golazy.dev/lazyview"
)

// Kind describes the browser worker API a script is intended for.
type Kind string

const (
	// ServiceWorker is registered through navigator.serviceWorker.
	ServiceWorker Kind = "service_worker"
	// WebWorker is constructed through new Worker.
	WebWorker Kind = "web_worker"
	// SharedWorker is constructed through new SharedWorker.
	SharedWorker Kind = "shared_worker"
)

// ScriptType describes how the browser should interpret the worker script.
type ScriptType string

const (
	// ClassicScript is the browser default worker script type.
	ClassicScript ScriptType = "classic"
	// ModuleScript loads the worker as an ES module.
	ModuleScript ScriptType = "module"
)

// Source describes where a worker script comes from.
type Source string

const (
	// SourceGenerated means the registry serves generated script bytes.
	SourceGenerated Source = "generated"
	// SourceAsset means another asset handler serves the script path.
	SourceAsset Source = "asset"
	// SourceRoute means an application route serves the script path.
	SourceRoute Source = "route"
)

// Worker describes one registered browser worker.
type Worker struct {
	Name        string     `json:"name"`
	Kind        Kind       `json:"kind"`
	Path        string     `json:"path"`
	Scope       string     `json:"scope,omitempty"`
	Type        ScriptType `json:"type,omitempty"`
	Source      Source     `json:"source"`
	Description string     `json:"description,omitempty"`
	ContentType string     `json:"content_type,omitempty"`
	Generated   bool       `json:"generated,omitempty"`
	PWA         bool       `json:"pwa,omitempty"`
}

// Manifest is the lazydev-visible worker inventory.
type Manifest struct {
	Workers []Worker `json:"workers"`
}

// Registry owns registered worker metadata and generated script handlers.
type Registry struct {
	workers  map[string]Worker
	paths    map[string]string
	handlers map[string]http.Handler
}

// Option configures a registered worker.
type Option func(*Worker)

// WithScope sets a service-worker scope.
func WithScope(scope string) Option {
	return func(worker *Worker) {
		worker.Scope = scope
	}
}

// WithScriptType sets the worker script type.
func WithScriptType(scriptType ScriptType) Option {
	return func(worker *Worker) {
		worker.Type = scriptType
	}
}

// WithSource records where the worker script is served from.
func WithSource(source Source) Option {
	return func(worker *Worker) {
		worker.Source = source
	}
}

// WithDescription records human-readable worker purpose for tooling.
func WithDescription(description string) Option {
	return func(worker *Worker) {
		worker.Description = description
	}
}

// WithContentType sets the generated script content type.
func WithContentType(contentType string) Option {
	return func(worker *Worker) {
		worker.ContentType = contentType
	}
}

// WithPWA marks a worker as owned by lazypwa.
func WithPWA() Option {
	return func(worker *Worker) {
		worker.PWA = true
	}
}

// New creates an empty worker registry.
func New() *Registry {
	return &Registry{
		workers:  map[string]Worker{},
		paths:    map[string]string{},
		handlers: map[string]http.Handler{},
	}
}

// AddScript registers and serves generated worker script bytes.
func (r *Registry) AddScript(name string, kind Kind, workerPath string, script []byte, options ...Option) error {
	if r == nil {
		return fmt.Errorf("lazyworkers: registry is nil")
	}
	content := append([]byte(nil), script...)
	worker := Worker{
		Name:        name,
		Kind:        kind,
		Path:        workerPath,
		Source:      SourceGenerated,
		Type:        ClassicScript,
		ContentType: "text/javascript; charset=utf-8",
		Generated:   true,
	}
	for _, option := range options {
		option(&worker)
	}
	if worker.ContentType == "" {
		worker.ContentType = "text/javascript; charset=utf-8"
	}
	return r.add(worker, generatedScriptHandler(worker.ContentType, content))
}

// AddHandler registers a worker served by handler.
func (r *Registry) AddHandler(name string, kind Kind, workerPath string, handler http.Handler, options ...Option) error {
	if r == nil {
		return fmt.Errorf("lazyworkers: registry is nil")
	}
	if handler == nil {
		return fmt.Errorf("lazyworkers: handler is nil")
	}
	worker := Worker{
		Name:   name,
		Kind:   kind,
		Path:   workerPath,
		Source: SourceRoute,
		Type:   ClassicScript,
	}
	for _, option := range options {
		option(&worker)
	}
	return r.add(worker, handler)
}

// AddAsset records a worker script served by the application's asset registry.
func (r *Registry) AddAsset(name string, kind Kind, workerPath string, options ...Option) error {
	if r == nil {
		return fmt.Errorf("lazyworkers: registry is nil")
	}
	worker := Worker{
		Name:   name,
		Kind:   kind,
		Path:   workerPath,
		Source: SourceAsset,
		Type:   ClassicScript,
	}
	for _, option := range options {
		option(&worker)
	}
	return r.add(worker, nil)
}

func (r *Registry) add(worker Worker, handler http.Handler) error {
	worker.Name = strings.TrimSpace(worker.Name)
	if worker.Name == "" {
		return fmt.Errorf("lazyworkers: worker name is required")
	}
	if _, ok := r.workers[worker.Name]; ok {
		return fmt.Errorf("lazyworkers: worker %q is already registered", worker.Name)
	}
	worker.Kind = normalizeKind(worker.Kind)
	if worker.Kind == "" {
		return fmt.Errorf("lazyworkers: worker %q kind is required", worker.Name)
	}
	worker.Path = normalizeWorkerPath(worker.Path)
	if worker.Path == "" {
		return fmt.Errorf("lazyworkers: worker %q path is required", worker.Name)
	}
	if existing := r.paths[worker.Path]; existing != "" {
		return fmt.Errorf("lazyworkers: worker path %q is already registered by %q", worker.Path, existing)
	}
	if worker.Type == "" {
		worker.Type = ClassicScript
	}
	if worker.Source == "" {
		if handler != nil {
			worker.Source = SourceRoute
		} else {
			worker.Source = SourceAsset
		}
	}
	if worker.Kind == ServiceWorker && worker.Scope == "" {
		worker.Scope = "/"
	}
	if worker.ContentType == "" && handler != nil {
		worker.ContentType = "text/javascript; charset=utf-8"
	}
	r.workers[worker.Name] = worker
	r.paths[worker.Path] = worker.Name
	if handler != nil {
		r.handlers[worker.Path] = handler
	}
	return nil
}

func normalizeKind(kind Kind) Kind {
	switch kind {
	case ServiceWorker, WebWorker, SharedWorker:
		return kind
	default:
		return ""
	}
}

func normalizeWorkerPath(workerPath string) string {
	workerPath = strings.TrimSpace(workerPath)
	if workerPath == "" {
		return ""
	}
	if !strings.HasPrefix(workerPath, "/") {
		workerPath = "/" + workerPath
	}
	return path.Clean(workerPath)
}

// Path returns the registered URL path for name.
func (r *Registry) Path(name string) (string, error) {
	worker, ok := r.Worker(name)
	if !ok {
		return "", fmt.Errorf("lazyworkers: worker %q not found", name)
	}
	return worker.Path, nil
}

// Worker returns a registered worker by name.
func (r *Registry) Worker(name string) (Worker, bool) {
	if r == nil {
		return Worker{}, false
	}
	worker, ok := r.workers[strings.TrimSpace(name)]
	return worker, ok
}

// Manifest returns a stable snapshot of all registered workers.
func (r *Registry) Manifest() Manifest {
	if r == nil {
		return Manifest{}
	}
	names := make([]string, 0, len(r.workers))
	for name := range r.workers {
		names = append(names, name)
	}
	sort.Strings(names)
	manifest := Manifest{Workers: make([]Worker, 0, len(names))}
	for _, name := range names {
		manifest.Workers = append(manifest.Workers, r.workers[name])
	}
	return manifest
}

// Empty reports whether no workers are registered.
func (r *Registry) Empty() bool {
	return r == nil || len(r.workers) == 0
}

// MiddlewareName returns the dispatcher-visible middleware name.
func (r *Registry) MiddlewareName() string {
	return "lazyworkers.Registry"
}

// Handler serves generated worker scripts and falls through to next for misses.
func (r *Registry) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		handler, ok := r.handlers[req.URL.Path]
		if !ok {
			next.ServeHTTP(w, req)
			return
		}
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		handler.ServeHTTP(w, req)
	})
}

// ServeHTTP serves generated worker scripts as a standalone handler.
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Handler(http.NotFoundHandler()).ServeHTTP(w, req)
}

func generatedScriptHandler(contentType string, script []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "no-cache")
		if req.Method == http.MethodHead {
			return
		}
		_, _ = w.Write(script)
	})
}

// Helpers returns lazyview-compatible worker helpers.
func (r *Registry) Helpers() map[string]any {
	return map[string]any{
		"worker_path": func(name string) (string, error) {
			return r.Path(name)
		},
		"service_worker_register": func(name string) (lazyview.Fragment, error) {
			worker, ok := r.Worker(name)
			if !ok {
				return lazyview.Fragment{}, fmt.Errorf("lazyworkers: worker %q not found", name)
			}
			if worker.Kind != ServiceWorker {
				return lazyview.Fragment{}, fmt.Errorf("lazyworkers: worker %q is not a service worker", name)
			}
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        serviceWorkerRegisterScript(worker),
			}, nil
		},
		"worker_script": func(name string) (lazyview.Fragment, error) {
			worker, ok := r.Worker(name)
			if !ok {
				return lazyview.Fragment{}, fmt.Errorf("lazyworkers: worker %q not found", name)
			}
			return lazyview.Fragment{
				ContentType: "text/html; charset=utf-8",
				Body:        workerConstructorScript(worker),
			}, nil
		},
	}
}

func serviceWorkerRegisterScript(worker Worker) string {
	pathJSON := jsonString(worker.Path)
	scopeJSON := jsonString(firstNonEmpty(worker.Scope, "/"))
	typeJSON := jsonString(firstNonEmpty(string(worker.Type), string(ClassicScript)))
	return `<script type="module">` +
		`if("serviceWorker"in navigator){window.addEventListener("load",()=>{navigator.serviceWorker.register(` +
		pathJSON + `,{scope:` + scopeJSON + `,type:` + typeJSON + `})})}` +
		`</script>`
}

func workerConstructorScript(worker Worker) string {
	nameJSON := jsonString(worker.Name)
	pathJSON := jsonString(worker.Path)
	typeJSON := jsonString(firstNonEmpty(string(worker.Type), string(ClassicScript)))
	constructor := "Worker"
	if worker.Kind == SharedWorker {
		constructor = "SharedWorker"
	}
	if worker.Kind == ServiceWorker {
		return serviceWorkerRegisterScript(worker)
	}
	return `<script type="module">` +
		`window.lazyworkers=window.lazyworkers||{};` +
		`window.lazyworkers[` + nameJSON + `]=()=>new ` + constructor + `(` + pathJSON + `,{type:` + typeJSON + `});` +
		`</script>`
}

func jsonString(value string) string {
	data, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
