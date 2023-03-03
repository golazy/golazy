package golazy

import (
	"io"

	"golazy.dev/lazyview/components/turbo"
	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/page"
	"golazy.dev/lazyview/static_files"
)

// Layout is the layout for the portal
type Layout struct {
	page.Page
}

func (l *Layout) RenderLayout(a *static_files.Manager, content []byte) io.WriterTo {
	l.Files = a
	l.Charset = "utf-8"
	l.Title = "GoLazy"
	l.Viewport = "width=device-width, initial-scale=1"
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
