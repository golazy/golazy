package lazyplugin

import (
	"github.com/golazy/golazy/lazycontext"
)

var Plugins = []Plugin{}

type Plugin interface {
	Name() string
	Desc() string
	URL() string
	Init(lazycontext.LazyContext)
}
