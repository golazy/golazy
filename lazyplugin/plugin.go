package lazyplugin

import (
	"golazy.dev/lazycontext"
)

var Plugins = []Plugin{}

type Plugin interface {
	Name() string
	Desc() string
	URL() string
	Init(lazycontext.AppContext)
}
