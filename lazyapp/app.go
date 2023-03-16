package lazyapp

import (
	"context"
	"net/http"

	"github.com/spf13/viper"
	"golazy.dev/lazyaction"
	"golazy.dev/lazyassets"
)

var Current *App

type App struct {
	Name        string
	Addr        string
	Dispatcher  lazyaction.Dispatcher
	Server      http.Server
	Assets      *lazyassets.Assets
	MiddleWares []Middleware
	h           http.Handler
}

func (a *App) Routes() []lazyaction.Route {
	return a.Dispatcher.Routes()
}

func (a *App) Middleware(f func(http.Handler) http.Handler) {
	a.MiddleWares = append(a.MiddleWares, f)
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}

func (a *App) Route(route_def string, target any) {
	a.Dispatcher.Assets = a.Assets
	a.Dispatcher.Route(route_def, target)
}
func (a *App) With(c lazyaction.Constraints) *lazyaction.Constraints {
	a.Dispatcher.Assets = a.Assets
	return a.Dispatcher.With(c)
}

func (a *App) PermanentRedirect(route_def string, target string) {
	a.Dispatcher.Route(route_def, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusMovedPermanently)
	})
}

func (a *App) Redirect(route_def string, target string) {
	a.Dispatcher.Route(route_def, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	})
}

func (a *App) Resource(resource any, opts ...*lazyaction.ResourceOptions) {
	a.Dispatcher.Assets = a.Assets
	a.Dispatcher.Resource(resource, opts...)
}

func (a *App) Init() {
	a.Dispatcher.Assets = a.Assets

	a.h = &a.Dispatcher

	// Add files
	if a.Assets != nil {
		a.h = a.Assets.NewMiddleware(a.h)
	}

	// Add logger
	a.h = loggerMiddleware(a.h)

	// Add panic handler
	a.h = panicMiddleware(a.h)

	// Add middlewares
	for _, m := range a.MiddleWares {
		a.h = m(a.h)
	}
}

func (a *App) Boot() {
	a.Init()

	viper.SetConfigFile(".env")
	viper.ReadInConfig()

	rootCmd(a).Execute()
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.h.ServeHTTP(w, r)
}
