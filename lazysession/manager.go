package lazysession

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
)

const defaultSessionName = "lazy_session"

type managerContextKey struct{}

// Config describes the default session manager created by lazyapp.
//
// Store is optional. When it is nil, Key and KeyPairs are used to build a
// CookieStore.
// Supplying Store keeps the lazyapp integration open to other backends without
// changing application configuration later.
type Config struct {
	Name  string
	Store Store
	// Key is the common app-facing session secret. It is deterministically
	// expanded with SHA-256 before constructing the cookie store.
	Key      string
	KeyPairs [][]byte
	Options  *Options
}

// Enabled reports whether this config should install session middleware.
func (c Config) Enabled() bool {
	return c.Store != nil || c.Key != "" || len(c.KeyPairs) > 0
}

// NewManager creates a request session manager from config.
func NewManager(config Config) (*Manager, error) {
	if !config.Enabled() {
		return nil, fmt.Errorf("lazysession: store, key, or key pairs are required")
	}

	name := config.Name
	if name == "" {
		name = defaultSessionName
	}
	if !isCookieNameValid(name) {
		return nil, fmt.Errorf("lazysession: invalid session name %q", name)
	}

	store := config.Store
	if store == nil {
		keyPairs := make([][]byte, 0, len(config.KeyPairs)+1)
		if config.Key != "" {
			keyPairs = append(keyPairs, deriveKey(config.Key))
		}
		for _, key := range config.KeyPairs {
			keyPairs = append(keyPairs, append([]byte(nil), key...))
		}
		store = NewCookieStore(keyPairs...)
	}
	applyOptions(store, config.Options)

	return &Manager{name: name, store: store}, nil
}

func applyOptions(store Store, options *Options) {
	if options == nil {
		return
	}
	copied := *options
	switch s := store.(type) {
	case *CookieStore:
		s.Options = &copied
		s.MaxAge(copied.MaxAge)
	case *FilesystemStore:
		s.Options = &copied
		s.MaxAge(copied.MaxAge)
	}
}

func deriveKey(key string) []byte {
	sum := sha256.Sum256([]byte(key))
	return sum[:]
}

// Manager provides request helpers and middleware for one application session.
type Manager struct {
	name  string
	store Store
}

// Name returns the cookie/session name used by this manager.
func (m *Manager) Name() string {
	return m.name
}

// Store returns the underlying session store.
func (m *Manager) Store() Store {
	return m.store
}

// Get returns the manager's default session for r.
func (m *Manager) Get(r *http.Request) (*Session, error) {
	if m == nil {
		return nil, fmt.Errorf("lazysession: manager is nil")
	}
	return m.store.Get(r, m.name)
}

// Read returns the manager's default session for r without marking it for save.
func (m *Manager) Read(r *http.Request) (*Session, error) {
	if m == nil {
		return nil, fmt.Errorf("lazysession: manager is nil")
	}
	return GetRegistry(r).Read(m.store, m.name)
}

// MarkDirty marks session as changed so the manager middleware saves it.
func (m *Manager) MarkDirty(r *http.Request, session *Session) error {
	if m == nil {
		return fmt.Errorf("lazysession: manager is nil")
	}
	return GetRegistry(r).MarkDirty(m.store, m.name, session)
}

// Handler installs m into the request context and saves registered sessions
// before the response is sent.
func (m *Manager) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(WithManager(r.Context(), m))
		GetRegistry(r)
		saver := &responseSaver{ResponseWriter: w, request: r}
		next.ServeHTTP(saver, r)
		saver.save()
	})
}

type responseSaver struct {
	http.ResponseWriter
	request *http.Request
	saved   bool
}

func (w *responseSaver) WriteHeader(status int) {
	w.save()
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseSaver) Write(data []byte) (int, error) {
	w.save()
	return w.ResponseWriter.Write(data)
}

func (w *responseSaver) save() {
	if w.saved {
		return
	}
	w.saved = true
	if err := Save(w.request, w.ResponseWriter); err != nil {
		panic(err)
	}
}

func (w *responseSaver) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseSaver) StartStream(status int) (http.ResponseWriter, error) {
	w.save()
	if starter, ok := w.ResponseWriter.(interface {
		StartStream(int) (http.ResponseWriter, error)
	}); ok {
		return starter.StartStream(status)
	}
	w.ResponseWriter.WriteHeader(status)
	return w.ResponseWriter, nil
}

// WithManager stores manager in ctx.
func WithManager(ctx context.Context, manager *Manager) context.Context {
	return context.WithValue(ctx, managerContextKey{}, manager)
}

// ManagerFromContext returns the session manager stored in ctx.
func ManagerFromContext(ctx context.Context) (*Manager, bool) {
	manager, ok := ctx.Value(managerContextKey{}).(*Manager)
	return manager, ok && manager != nil
}

// Get returns the configured application session from r's context.
func Get(r *http.Request) (*Session, error) {
	manager, ok := ManagerFromContext(r.Context())
	if !ok {
		return nil, fmt.Errorf("lazysession: manager is missing from request context")
	}
	return manager.Get(r)
}

// Read returns the configured application session without marking it for save.
func Read(r *http.Request) (*Session, error) {
	manager, ok := ManagerFromContext(r.Context())
	if !ok {
		return nil, fmt.Errorf("lazysession: manager is missing from request context")
	}
	return manager.Read(r)
}

// MarkDirty marks session as changed so it is saved before the response is sent.
func MarkDirty(r *http.Request, session *Session) error {
	manager, ok := ManagerFromContext(r.Context())
	if !ok {
		return fmt.Errorf("lazysession: manager is missing from request context")
	}
	return manager.MarkDirty(r, session)
}
