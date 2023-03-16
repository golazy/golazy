package assets

import (
	"io"
	"portal/assets"
	"portal/components/table"
	"portal/layouts/golazy"
)

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index() io.WriterTo {

	return table.New(assets.Manager.Routes())
}
