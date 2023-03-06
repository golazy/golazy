package http

import (
	"io"

	"portal/layouts/golazy"

	"golazy.dev/lazyview/html"
)

type Controller struct {
	golazy.Layout
}

func (h *Controller) Index() io.WriterTo {

	return html.H1("pepe")

}
