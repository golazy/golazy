package builds

import (
	"io"
	"portal/layouts/golazy"
	pe "portal/resources/events"

	"golazy.dev/lazyroom"
	"golazy.dev/lazyview/html"
)

func init() {
	startC := make(chan (lazyroom.Event))
	errC := make(chan (lazyroom.Event))
	pe.Subscribe(startC, "devapp/build_start")
	pe.Subscribe(errC, "devapp/build_error")

	go func() {
		for {
			select {
			case <-startC:
				Data = []byte("Building...")
			case e := <-errC:
				Data = e.Data
			}
		}
	}()
}

var Data []byte

type Controller struct {
	golazy.Layout
}

func (c *Controller) Index() string {
	return "Builds"
}

func (c *Controller) GetRerouter() io.WriterTo {
	return html.Code(
		html.Pre(
			string(Data),
		),
	)
}
