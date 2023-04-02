package assets

import (
	"embed"

	"golazy.dev/lazyassets"
)

//go:embed public
var FS embed.FS

var (
	Manager    = lazyassets.New()
	Assets     = Manager.AddFS(FS, "public")
	Stylesheet = Assets.NewStylesheet("app.css")
)
