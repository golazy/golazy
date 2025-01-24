package lazycontroller

import (
	"net/http"

	"golazy.dev/lazyview"
)

func (c *Base) Render(opts *lazyview.Options) error {
	c.bodySent = true
	if opts == nil {
		opts = &lazyview.Options{}
	}
	if opts.Ctx == nil {
		opts.Ctx = c.R.Context()
	}
	if opts.Writer == nil {
		opts.Writer = c.W
	}
	if opts.Action == "" {
		opts.Action = c.Route.Action
	}
	if opts.Controller == "" {
		opts.Controller = c.Route.Controller
	}
	if opts.Namespace == "" {
		opts.Namespace = c.Route.Namespace
	}
	if opts.Accept == "" {
		opts.Accept = c.R.Header.Get("Accept")
	}
	if opts.Layout == "" {
		opts.Layout = c.Layout
	}
	opts.UseLayout = true

	opts.Variables = c.viewVars

	err := c.Views.Render(*opts)
	if err != nil {
		http.Error(c.W, err.Error(), http.StatusInternalServerError)
	}
	return err
}

func (c *Base) RenderContent(content string) error {
	return c.Render(&lazyview.Options{
		Content: content,
	})
}
