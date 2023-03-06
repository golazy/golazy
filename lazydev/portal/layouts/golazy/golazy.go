package golazy

import (
	"io"

	lazyassets "golazy.dev/lazyassets"
	"golazy.dev/lazyview/components/turbo"
	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/page"
	"golazy.dev/lazyview/style"
)

func init() {
}

// Layout is the layout for the portal
type Layout struct {
	page.Page
}

func (l *Layout) RenderLayout(a *lazyassets.Manager, content []byte) io.WriterTo {

	l.Files = a
	l.Charset = "utf-8"
	l.Title = "GoLazy"
	l.Viewport = "width=device-width, initial-scale=1"
	l.Styles = append(l.Styles, style.Style{
		Content: `

		`,
	})

	l.Content = Body(
		H1(
			Img(Src(a.Get("img/logo.svg")), Alt("GoLazy")),
		),
		Main(
			content,
		),
	)
	l.Use(turbo.Component)

	return l.Element()
}
