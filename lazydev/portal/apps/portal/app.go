package portal

import (
	"portal/assets"
	"portal/resources/devapp"

	"golazy.dev/lazyaction"
	lazyassets "golazy.dev/lazyassets"
	"golazy.dev/lazyview/component"
)

var App = &PortalApp{}

func init() {

	component.InstallAll(component.InstallOptions{
		Path:  "../../assets/public",
		Cache: "../../assets/cache",
	})

	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "http", Path: "/"})
	//App.Router.Resource(&rhttp.HttpController{}, &lazyaction.ResourceOptions{Scheme: "https", Path: "/"})

	App.Files = lazyassets.NewManager(assets.FS, "public")
	App.Resource(&devapp.Controller{}, &lazyaction.ResourceOptions{Path: "/"})

}
