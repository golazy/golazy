package lazydev

import (
	_ "embed"
	"io"
	"net/http"
	"net/url"
	"os"

	. "golazy.dev/lazyview/html"
	"golazy.dev/lazyview/layout"
	"golazy.dev/lazyview/layout/lazylayout"
)

//go:embed http.js
var script string

var pageLayout layout.LayoutTemplate = *lazylayout.Layout

func init() {
	pageLayout.Scripts = append(pageLayout.Scripts, script)
}

func httpHandler(pem, key string) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/golazy/commands", control)
	mux.HandleFunc("/golazy/ca.pem", func(w http.ResponseWriter, r *http.Request) {

		f, err := os.Open(certPEM())
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		url := &url.URL{}
		url.Host = r.Host
		url.Scheme = "https"
		url.Path = r.URL.Path
		url.RawQuery = r.URL.RawQuery

		content := []any{
			lazylayout.PageHeader(),
			lazylayout.PageNav(),
			Main(
				H1("Welcome to lazygo"),
				P("Lazygo is the fastest way to develop apps in go."),
				P("Let’s setup your machine first."),
				P("As lazygo uses secure connections by default, you need first to install your lazygo certificate authority."),
				P(A(Class("button blue"), Href("/golazy/ca.pem"), Download(), "Downlaod CA")),
				P("This are the installation instructions for the CA:",
					Ul(
						Li(A(Href("#windows"), "How to install on Linux")),
						Li(A(Href("#windows"), "How to install on Windows")),
						Li(A(Href("#windows"), "How to install on Mac")),
					),
				),
				A(Class("button red"), Href(url.String()), Download(), "Go to the secure site"),
			),
		}

		pageLayout.With(content).WriteTo(w)
	})

	return mux
}
