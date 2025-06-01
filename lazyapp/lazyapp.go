// Package golazy is the go framework or building web applications
package lazyapp

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/golazy/golazy/layerfs"
	"github.com/golazy/golazy/lazyassets"
	"github.com/golazy/golazy/lazycontext"
	"github.com/golazy/golazy/lazydispatch"
	"github.com/golazy/golazy/lazyenv"
	"github.com/golazy/golazy/lazyhttp"
	"github.com/golazy/golazy/lazyplugin"
	"github.com/golazy/golazy/lazyservice"
	"github.com/golazy/golazy/lazyview"
	"github.com/golazy/golazy/lazyview/engines/raw"
	"github.com/golazy/golazy/lazyview/engines/tpl"
)

var DefaultMiddlewares = []func(http.Handler) http.Handler{}

// GoLazyApp it's the glue of all golazy modules
type GoLazyApp struct {
	LazyService       lazyservice.Manager
	LazyHTTP          lazyhttp.HTTPService
	LazyAssets        lazyassets.Server
	LazyView          lazyview.Views
	LazyDispatch      *lazydispatch.Dispatcher
	LazyPlugins       []lazyplugin.Plugin
	DisableInterrupts bool
	onContextCreate   []hook
}

type hook struct {
	caller string
	fn     func(context.Context)
}

var nameKey = "app.name"
var versionKey = "app.version"

type Config struct {
	PublicFS    fs.FS
	ViewsFS     fs.FS
	Helpers     map[string]any
	Plugins     []lazyplugin.Plugin
	BeforeStart []func(context.Context)
}

// New creates a new GolazyApp instance
// See [golazy.dev/golazy/lazyapp.New]
func New(name, version string) *GoLazyApp {
	ctx := lazycontext.New()
	ctx.AddValue(nameKey, name)
	ctx.AddValue(versionKey, version)
	app := &GoLazyApp{
		LazyService: lazyservice.New(),
	}
	app.init()
	return app
}

func callerInfo(n int) string {
	_, file, line, ok := runtime.Caller(n)
	if !ok {
		return "Could not get caller information"
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// BeforeStart adds a function to be called before the app starts
func (app *GoLazyApp) BeforeStart(fn func(context.Context)) {

	app.onContextCreate = append(app.onContextCreate, hook{fn: fn, caller: callerInfo(2)})

}

func (app *GoLazyApp) init() *GoLazyApp {

	// Views
	app.LazyView.Engines = map[string]lazyview.Engine{
		"tpl": &tpl.Engine{},
		"txt": &raw.Engine{},
	}
	app.LazyView.FS = layerfs.New()

	// Dispatcher
	app.LazyDispatch = lazydispatch.New()

	// Assets
	app.LazyAssets.Storage = &lazyassets.Storage{}
	app.LazyAssets.Handler = app.LazyDispatch

	// Server
	app.LazyHTTP.Addr = lazyenv.Addr()
	app.LazyHTTP.Handler = &app.LazyAssets
	app.LazyService.AddService(&app.LazyHTTP)

	// Expose services to the context
	app.BeforeStart(func(ctx context.Context) {
		lazycontext.Set(ctx, app.LazyService)
		lazycontext.Set(ctx, &app.LazyView)
		lazycontext.Set(ctx, &app.LazyHTTP)
		lazycontext.Set(ctx, app.LazyDispatch)
		lazycontext.Set(ctx, app.LazyAssets.Storage)
		lazycontext.Set(ctx, &app.LazyAssets)
	})

	// Enable helpers
	app.LazyView.Helpers = map[string]any{
		"path_for": func(args ...any) string {
			return app.LazyDispatch.PathFor(args...)
		},
		"permalink": func(path string) string {
			f := app.LazyAssets.Find(path)
			if f == nil {
				panic(fmt.Sprintf("Asset not found: %s", path))
			}
			return f.Permalink()
		},
	}

	// Add default middleware
	for _, m := range DefaultMiddlewares {
		app.Use(m)
	}

	return app
}

// AddService adds a service to the app
// See [golazy.dev/golazy/lazyservice.AddService]
func (app *GoLazyApp) AddService(srv lazyservice.Service) {
	app.LazyService.AddService(srv)
}

// AddPlugin adds a plugin to the app
func (app *GoLazyApp) AddPlugin(plugin lazyplugin.Plugin) {
	app.LazyPlugins = append(app.LazyPlugins, plugin)
}

// Draw draws the http routes of the server
// See [golazy.dev/golazy/lazydispatch.Dispatcher.Draw]
func (app *GoLazyApp) Draw(fn func(r *lazydispatch.Scope)) *lazydispatch.Scope {
	return app.LazyDispatch.Draw(fn)
}

func (app *GoLazyApp) AddHelpers(helpers map[string]any) {
	for k, v := range helpers {
		app.LazyView.Helpers[k] = v
	}
}

// Use adds a middleware to the server
// See [golazy.dev/golazy/lazydispatch.Dispatcher.Use]
func (app *GoLazyApp) Use(middleware func(http.Handler) http.Handler) {
	app.LazyDispatch.Use(middleware)
}

// Public adds all the public files and assets
// It expects all the public files to be in a directory called "public"
// See [golazy.dev/golazy/lazyassets.Storage.AddFS]
func (app *GoLazyApp) Public(fs fs.FS) {
	app.LazyAssets.AddFS(dir(fs, "public"))
}

// Views adds all the views
// It expects all the views to be in a directory called "views"
// See [golazy.dev/golazy/lazyview.Views] and [golazy.dev/golazy/layerfs.FS.Add]
func (app *GoLazyApp) Views(fs fs.FS) {
	app.LazyView.FS.(*layerfs.FS).Add(dir(fs, "views"))
}

// Run will start the application
// See [golazy.dev/golazy/lazyservice.Manager.Run]
func (app *GoLazyApp) Run(ctx context.Context) error {
	if !app.DisableInterrupts {
		intCtx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
		defer cancel()
		ctx = intCtx
	}
	lctx := lazycontext.NewWithContext(ctx)
	lazycontext.Set(lctx, app)

	for _, hook := range app.onContextCreate {
		slog.Info("BeforeStart", "caller", hook.caller)
		hook.fn(lctx)
	}

	// Initialize plugins
	for _, plugin := range app.LazyPlugins {
		plugin.Init(lctx)
	}

	return app.LazyService.Run(lctx)
}

// Start will start the application with Run and return a channel with the error
func (app *GoLazyApp) Start(ctx context.Context) <-chan (error) {
	errCh := make(chan (error))
	go func() {
		errCh <- app.Run(ctx)
	}()
	runtime.Gosched()
	return errCh

}

func dir(files fs.FS, dir string) fs.FS {
	files, err := fs.Sub(files, dir)
	if err != nil {
		fmt.Printf("Error: Subdirectory %s not found: %s\n", dir, err.Error())
		os.Exit(-1)
	}
	return files
}
