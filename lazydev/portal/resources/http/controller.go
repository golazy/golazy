package http

import (
	"io"

	"golazy.dev/lazyview/html"
)

type HttpController struct {
}

func (h *HttpController) Index() io.WriterTo {

	return html.H1("Hello World!")

}
