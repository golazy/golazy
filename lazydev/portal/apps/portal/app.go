package portal

import (
	"bytes"
	"portal/apps/portal/rerouter"
	"portal/assets"
	ar "portal/resources/assets"
	"portal/resources/builds"
	"portal/resources/components"
	"portal/resources/deploys"
	"portal/resources/devapp"
	"portal/resources/docs"
	"portal/resources/editor"
	"portal/resources/http"
	"portal/resources/routes"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydev/injector"
	"golazy.dev/lazyview/component"
)

var App = &PortalApp{
	App: lazyapp.App{
		Assets: assets.Assets,
	},
}

// EmptyGif in decimal
var EmptyGif = []byte{
	71, 73, 70, 56, 57, 97, 1, 0, 1, 0, 128, 0, 0, 255, 255, 255, 0, 0, 0, 44, 0, 0, 0, 0, 1, 0, 1, 0, 0, 2, 2, 68, 1, 0, 59,
}

func init() {

	//d, err := os.Getwd()
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("apps/portal/app.go Running in ", d)

	//	component.InstallAll(component.InstallOptions{
	//		Path:  filepath.Join(d, "assets/public"),
	//		Cache: filepath.Join(d, "assets/cache"),
	//	})

	component.DefaultInstallOptions = component.InstallOptions{
		Path:  "assets/public",
		Cache: "assets/cache",
	}

	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "http", Path: "/"})
	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "https", Path: "/"})
	App.Middleware(injector.New(bytes.NewBufferString("<!-- holasdfasdfa -->")))
	App.Middleware(rerouter.New)

	h := App.With(lazyaction.Constraints{Scheme: "http"})
	h.Resource(&http.Controller{}, &lazyaction.ResourceOptions{Path: "/"})
	// Add ok handler for the https test
	App.Route("/golazy/http/ok", func() []byte { return EmptyGif })

	admin := App.With(lazyaction.Constraints{Scheme: "https", Prefix: "golazy"})

	admin.Resource(&devapp.Controller{})
	admin.Resource(&routes.Controller{})
	admin.Resource(&editor.Controller{})
	admin.Resource(&docs.Controller{})
	admin.Resource(&ar.Controller{})
	admin.Resource(&builds.Controller{})
	admin.Resource(&components.Controller{})
	admin.Resource(&deploys.Controller{})
	admin.Resource(&http.Controller{})

	// fmt.Println(App.Routes())
}
