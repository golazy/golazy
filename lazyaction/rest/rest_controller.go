package rest

import (
	"io"
	"net/http"

	"golazy.dev/lazysupport"
	"golazy.dev/lazyview/components"
	"golazy.dev/lazyview/components/table"
	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

type Options struct {
	Columns []string
	Storage Storage
	Model   any
}

func New(opts Options) Controller {
	if opts.Storage == nil {
		panic("missing Storage")
	}

	rc := Controller{
		opts: opts,
		name: opts.Storage.Name(),
	}

	if _, ok := opts.Storage.(StorageWriter); ok {
		rc.canWrite = true
	}

	if _, ok := opts.Storage.(StorageEraser); ok {
		rc.canDelete = true
	}

	rc.name = opts.Storage.Name()

	return rc
}

type Controller struct {
	canWrite  bool
	canDelete bool
	name      string
	opts      Options
}

func (rc *Controller) Index(w http.ResponseWriter, r *http.Request) (io.WriterTo, error) {
	rows, err := rc.opts.Storage.List()
	if err != nil {
		return nil, err
	}

	var newModel io.WriterTo
	if rc.canWrite {
		newModel = html.A(
			html.H1("New "+rc.opts.Storage.Name()),
			html.Href("new"),
		)
	}

	return nodes.Collection(
		html.H1(lazysupport.Pluralize(rc.opts.Storage.Name())),
		html.H1(lazysupport.Pluralize(rc.opts.Storage.Name())),
		html.H1(lazysupport.Pluralize(rc.opts.Storage.Name())),
		html.H1(lazysupport.Pluralize(rc.opts.Storage.Name())),
		newModel,
		table.New(rows, rc.opts.Columns),
	), nil
}

func (rc *Controller) New(w http.ResponseWriter, r *http.Request) (io.WriterTo, error) {
	if rc.opts.Storage == nil {
		panic("There is no storage for this controller")
	}
	return nodes.Collection(
		html.H1("New "+rc.name),
		components.Inspect(rc.opts.Storage.New()),
	), nil
}
