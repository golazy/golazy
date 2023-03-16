package logo

import (
	"portal/assets"

	_ "embed"

	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

//go:embed logo.svg
var logo []byte

func init() {
	assets.Assets.AddFile("img/logo.svg", logo)
}

const (
	SizeSmall  = "32 px"
	SizeLarge  = "64 px"
	SizeMedium = "48 px"
)

func Logo(size string) nodes.Element {
	return html.Img(
		html.Src(assets.Assets.Get("img/logo.svg")),
		html.Alt("GoLazy"),
		html.Width(size),
	)
}

func Small() nodes.Element {
	return Logo(SizeSmall)
}
