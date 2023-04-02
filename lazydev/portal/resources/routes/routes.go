package routes

import (
	"io"
	"portal/layouts/golazy"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyview/components/table"
)

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index(routes []lazyaction.Route) io.WriterTo {

	return table.New(routes)

}
