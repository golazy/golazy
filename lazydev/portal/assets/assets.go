package assets

import (
	"embed"

	"golazy.dev/lazyassets"
)

//go:embed public/*
var FS embed.FS

var Mangaer *lazyassets.Manager

func init() {

	Mangaer = lazyassets.NewManager(FS, "public")

}
