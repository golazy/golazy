package builds

import "portal/layouts/golazy"

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index() string {
	return "Builds"
}
