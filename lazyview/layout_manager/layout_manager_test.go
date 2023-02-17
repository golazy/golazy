package layout_manager

import (
	"embed"
	"testing"
)

type App struct {
}

type Mails struct {
}

type Domains struct {
}

//go:embed *.tpl
var views embed.FS

func (a *Domains) Index() {

	switch {
	default:
		fallthrough
	case ctx.Accepts("turbo"):
		fallthrough
	case ctx.Accepts("html"):
		return H1("html")
	case ctx.Accepts("json"):
		return AsJson("json")
	case ctx.Accepts("xml"):
		return AsXML("xml")
	case ctx.Accepts("yaml"):
		return AsXML("yaml")

	}
}

func TestLayoutM(t *testing.T) {

	lm := &LayoutManager{}

	lm.RegisterLayout(myLayout{})

}
