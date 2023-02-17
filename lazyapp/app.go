package lazyapp

import (
	"context"
	"net/http"

	"golazy.dev/lazyaction"
	"golazy.dev/lazydev"
)

// type App struct {
// 	LayoutManager layout_manager.LayoutManager
// 	AssetManager  asset_manager.asset_manager
// 	Router        lazyaction.Router
// }

type App struct {
	Name   string
	Router *lazyaction.Router
	Server *lazydev.Server
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.Router == nil {
		a.Router = &lazyaction.Router{}
	}
	a.Router.ServeHTTP(w, r)
}

func (a *App) Route(what ...any) {
	if a.Router == nil {
		a.Router = &lazyaction.Router{}
	}

	a.Router.Route(what...)
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.Server.Shutdown(ctx)
}
