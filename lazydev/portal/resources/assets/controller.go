package assets

import (
	"io"
	"portal/assets"
	"portal/layouts/golazy"

	"golazy.dev/lazyview/components/table"
)

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index() io.WriterTo {

	return table.New(assets.Manager.Routes())
}
