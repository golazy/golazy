package lazyapp

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyroutes"
)

type Config struct {
	Name        string
	Drawer      func(*lazyroutes.Scope)
	Public      func() (fs.FS, error)
	Views       func() (fs.FS, error)
	Context     func(context.Context) context.Context
	Helpers     []map[string]any
	Middlewares []lazydispatch.Middleware
}

type App struct {
	Name       string
	Context    context.Context
	Dispatcher *lazydispatch.Dispatcher
	Router     *lazyroutes.Scope
}

var afterDraw = func(*lazyroutes.Scope) {}

func New(config Config) *App {
	ctx := context.Background()
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

	router := lazyroutes.New(ctx)
	if config.Drawer != nil {
		config.Drawer(router)
	}
	afterDraw(router)
	if renderer != nil {
		renderer.AddHelpers(router.RegisterHelpers())
		for _, helpers := range config.Helpers {
			renderer.AddHelpers(helpers)
		}
	}

	dispatcher := lazydispatch.NewDispatcher()
	dispatcher.Use(lazydispatch.RouteOnly(
		router,
		lazydispatch.ResponseBuffer(),
		lazydispatch.ETag(),
	))
	for _, middleware := range config.Middlewares {
		dispatcher.Use(middleware)
	}
	dispatcher.Use(lazydispatch.Router(router))
	if config.Public != nil {
		public, err := config.Public()
		if err != nil {
			panic(fmt.Errorf("open embedded public files: %w", err))
		}
		dispatcher.Use(lazydispatch.Public(public))
	}

	return &App{
		Name:       config.Name,
		Context:    ctx,
		Dispatcher: dispatcher,
		Router:     router,
	}
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app.Dispatcher.ServeHTTP(w, r)
}
