package lazyapp

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"golazy.dev/lazyassets"
	"golazy.dev/lazycontroller"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyroutes"
	"golazy.dev/lazysession"
)

type Config struct {
	Name         string
	Drawer       func(*lazyroutes.Scope)
	Public       func() (fs.FS, error)
	Views        func() (fs.FS, error)
	Context      func(context.Context) context.Context
	Helpers      []map[string]any
	Assets       []lazyassets.Source
	AssetOptions []lazyassets.Option
	Sessions     lazysession.Config
	Middlewares  []lazydispatch.Middleware
}

type App struct {
	Name       string
	Context    context.Context
	Dispatcher *lazydispatch.Dispatcher
	Router     *lazyroutes.Scope
	Assets     *lazyassets.Registry
	Sessions   *lazysession.Manager
}

var afterDraw = func(*lazyroutes.Scope) {}

func New(config Config) *App {
	ctx := context.Background()
	var sessions *lazysession.Manager
	if config.Sessions.Enabled() {
		var err error
		sessions, err = lazysession.NewManager(config.Sessions)
		if err != nil {
			panic(fmt.Errorf("initialize sessions: %w", err))
		}
		ctx = lazysession.WithManager(ctx, sessions)
	}

	var renderer *lazycontroller.Renderer
	if config.Views != nil {
		views, err := config.Views()
		if err != nil {
			panic(fmt.Errorf("open views: %w", err))
		}
		renderer, err = lazycontroller.NewRenderer(views)
		if err != nil {
			panic(fmt.Errorf("initialize renderer: %w", err))
		}
		ctx = lazycontroller.WithRenderer(ctx, renderer)
	}
	if config.Context != nil {
		ctx = config.Context(ctx)
	}

	assets := lazyassets.New(config.AssetOptions...)
	if config.Public != nil {
		public, err := config.Public()
		if err != nil {
			panic(fmt.Errorf("open embedded public files: %w", err))
		}
		ctx = lazycontroller.WithErrorPages(ctx, public)
		if err := assets.AddFS(public); err != nil {
			panic(fmt.Errorf("register public assets: %w", err))
		}
	}
	for _, source := range config.Assets {
		if source == nil {
			panic(fmt.Errorf("register generated assets: asset source is nil"))
		}
		if err := source.Assets(assets); err != nil {
			panic(fmt.Errorf("register generated assets: %w", err))
		}
	}

	router := lazyroutes.New(ctx)
	if config.Drawer != nil {
		config.Drawer(router)
	}
	afterDraw(router)
	if renderer != nil {
		renderer.AddHelpers(router.RegisterHelpers())
		renderer.AddHelpers(assets.Helpers())
		for _, helpers := range config.Helpers {
			renderer.AddHelpers(helpers)
		}
	}

	dispatcher := lazydispatch.NewDispatcher()
	dispatcher.Use(lazydispatch.RouteOnly(
		router,
		lazydispatch.ResponseBuffer(),
		lazydispatch.MiddlewareFunc(lazycontroller.ErrorHandler(ctx)),
		lazydispatch.ETag(),
	))
	if sessions != nil {
		dispatcher.Use(sessions)
	}
	for _, middleware := range config.Middlewares {
		dispatcher.Use(middleware)
	}
	dispatcher.Use(lazydispatch.Router(router))
	if !assets.Empty() {
		dispatcher.Use(lazydispatch.MiddlewareFunc(func(next http.Handler) http.Handler {
			return assets.Handler(next)
		}))
	}

	return &App{
		Name:       config.Name,
		Context:    ctx,
		Dispatcher: dispatcher,
		Router:     router,
		Assets:     assets,
		Sessions:   sessions,
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.Dispatcher.ServeHTTP(w, r)
}
