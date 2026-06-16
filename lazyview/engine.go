package lazyview

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"sync"
)

// Engine renders one view file using a concrete template implementation.
type Engine interface {
	Render(ctx *Context, writer io.Writer, file string) error
}

// CacheContext is passed to engines that can precompile templates.
type CacheContext struct {
	FS        fs.FS
	Extension string
	Helpers   map[string]any
}

// CacheableEngine can build or rebuild a template cache after app setup.
type CacheableEngine interface {
	Cache(ctx CacheContext) error
}

// CacheClearer can discard cached templates when view configuration changes.
type CacheClearer interface {
	ClearCache()
}

// EngineFactory creates a fresh engine instance for one Views value.
type EngineFactory func() Engine

var (
	engineRegistryMu sync.RWMutex
	engineRegistry   = map[string]EngineFactory{}
)

// RegisterEngine registers a template engine for a file extension.
//
// Extensions are matched without a leading dot. Engine packages normally call
// this from init, allowing applications to opt into engines with a blank import.
func RegisterEngine(extension string, factory EngineFactory) {
	extension = strings.TrimPrefix(strings.TrimSpace(extension), ".")
	if extension == "" {
		panic("lazyview: engine extension is required")
	}
	if factory == nil {
		panic("lazyview: engine factory is required")
	}

	engineRegistryMu.Lock()
	defer engineRegistryMu.Unlock()
	engineRegistry[extension] = factory
}

func registeredEngines() map[string]Engine {
	engineRegistryMu.RLock()
	defer engineRegistryMu.RUnlock()

	engines := make(map[string]Engine, len(engineRegistry))
	for extension, factory := range engineRegistry {
		engine := factory()
		if engine == nil {
			panic(fmt.Sprintf("lazyview: engine factory for %q returned nil", extension))
		}
		engines[extension] = engine
	}
	return engines
}
