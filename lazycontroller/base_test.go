package lazycontroller

import (
	"testing"

	"golazy.dev/memfs"
)

type HamburguersController struct {
	Base
}

func (c *HamburguersController) Index() {
}

func TestBase(t *testing.T) {

	memfs.New().AddMap(map[string]string{
		"layouts/application.html.tpl": "<html><body>{{.Content}}</body></html>",
		"hamburguers/index.html.tpl":   "<ul>{{ range .Hamburguers }}<li>{{ .Name }}</li>{{ end }}</ul>",
		"hamburguers/show.html.tpl":    "<h1>{{ .Hamburguer.Name }}</h1>",
	})

	//lazytest.New(ctx)

	//ctrl := lazytest.Controller(t, &HamburguersController{})
	//ctrl

}
