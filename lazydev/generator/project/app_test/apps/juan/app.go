package juan



import (
	"assets"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydev/injector"
	"golazy.dev/lazyview/component"
)

var App = &lazyapp.App{
    Name: "Juan",
    Assets: assets.Assets,
}

var Admin = App.With(lazyaction.Constraints{Prefix: "admin"})


func init() {

	component.DefaultInstallOptions = component.InstallOptions{
		Path:  "assets/public",
		Cache: "assets/cache",
	}

    App.Route("/", func() string { 
        return "hello world"
    })

    // App.Resource(&posts.Controller{}) to add a resource


    // App.Middleware(injector.New(bytes.NewBufferString(""))) to add a middleware


	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "http", Path: "/"})
	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "https", Path: "/"})
	App.Middleware(injector.New(bytes.NewBufferString("")))
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
