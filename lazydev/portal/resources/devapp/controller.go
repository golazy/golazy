package devapp

import (
	"io"
	"portal/layouts/golazy"

	. "golazy.dev/lazyview/html"
)

type Controller struct {
	golazy.Layout
}

func (a *Controller) Index() io.WriterTo {

	return Div(
		H1("Install certificate"),
		A(Href("/download_cert"), "Download certificate", Download()),
	)

}

func (a *Controller) Status() string {
	return "ok"
}
