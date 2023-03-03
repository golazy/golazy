package portal

import (
	"net/http"
	"net/url"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp"
	rhttp "golazy.dev/lazydev/internal/portal/resources/http"
	"golazy.dev/lazydev/server"
)

type PortalApp struct {
	lazyapp.App
	PortalDisabled bool
}

func (a *PortalApp) DisablePortal() {
	a.PortalDisabled = true
}

func (a *PortalApp) Open(url *url.URL) {
}
func (a *PortalApp) Close() {

}

var Portal = &PortalApp{
	App: lazyapp.App{
		Name: "portal",
	},
}

var App = &PortalApp{
	App: lazyapp.App{
		Name: "portal",
	},
}

func init() {
	App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Path: "http:///"})
	App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Path: "https:///"})
}

func (a *PortalApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.App.ServeHTTP(w, r)
}

func (a *PortalApp) Event(e server.Event) {

}
