package routes

import (
	"io"
	"portal/components/table"
	"portal/layouts/golazy"

	"golazy.dev/lazyaction"
)

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index(routes []lazyaction.Route) io.WriterTo {

	return table.New(routes)

}
