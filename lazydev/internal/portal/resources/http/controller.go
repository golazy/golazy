package http

import (
	"io"

	"golazy.dev/lazydev/internal/portal/layouts/golazy"
	"golazy.dev/lazyview/html"
)

type HttpController struct {
	golazy.Layout
}

func (h *HttpController) Index() io.WriterTo {

	return html.H1("pepe")

}
