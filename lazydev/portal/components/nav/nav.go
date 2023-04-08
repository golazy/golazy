package nav

import (
	"io"
	"net/url"
	"portal/assets"
	"portal/components/logo"

	_ "embed"

	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

//go:embed nav.css
var css string

func init() {
	assets.Stylesheet.Add(css)
}

func Navigation(current *url.URL) io.WriterTo {
	return Nav(Id("nav"),
		logo.Logo(logo.SizeMedium),
		Ul(
			Li(LinkUnlessCurrent(current, "/golazy/app", "App")),
			Li(LinkUnlessCurrent(current, "/golazy/routes", "Routes")),
			Li(LinkUnlessCurrent(current, "/golazy/editor", "Editor")),
			Li(LinkUnlessCurrent(current, "/golazy/docs", "Docs")),
			Li(LinkUnlessCurrent(current, "/golazy/assets", "Assets")),
			Li(LinkUnlessCurrent(current, "/golazy/builds", "Builds")),
			Li(LinkUnlessCurrent(current, "/golazy/components", "Components")),
			Li(LinkUnlessCurrent(current, "/golazy/deploys", "Deploys")),
			Li(LinkUnlessCurrent(current, "/golazy/new", "New")),
			Li(Class("spacer")),
			Li(Class("spacer")),
			Li(LinkUnlessCurrent(current, "/golazy/store", "Store")),
			Li(LinkUnlessCurrent(current, "/golazy/account", "Account")),
		),
	)
}

func LinkUnlessCurrent(current *url.URL, dest, text string, extra ...any) io.WriterTo {
	if isCurrent(current, dest) {
		return nodes.Text(text)
	}
	return A(Href(dest), text, extra)
}

func isCurrent(current *url.URL, dest string) bool {
	url, err := url.Parse(dest)
	if err != nil {
		return false
	}

	if url.Scheme != "" && url.Scheme != current.Scheme {
		return false
	}

	if url.Host != "" && url.Host != current.Host {
		return false
	}

	if url.Path != current.Path {
		return false
	}

	return true
}
