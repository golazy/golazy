package lazycontroller

import "io"

func (c *Controller) TurboStream(streams ...io.WriterTo) {
	c.Layout = ""
	c.W.Header().Set("Content-Type", "text/vnd.turbo-stream.html")
	for _, s := range streams {
		s.WriteTo(c.W)
	}

}
