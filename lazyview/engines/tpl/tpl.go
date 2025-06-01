package tpl

import (
	"context"
	"html/template"
	"io"
	"sync"

	"github.com/golazy/golazy/lazyview"
)

var (
	Extensions = []string{"tpl"}
)

type MissingKey int

const (
	MissingKeyZero MissingKey = iota
	MissingKeyError
	MissingKeyInvalid
)

type Engine struct {
	tplsL sync.RWMutex
	MissingKey
	tpls map[string](*template.Template)
}

var _ lazyview.Engine = &Engine{}

func (e *Engine) Render(ctx context.Context, views *lazyview.Views, w io.Writer, vars map[string]any, helpers map[string]any, file string) error {
	var err error
	e.tplsL.RLock()
	t, ok := e.tpls[file]
	e.tplsL.RUnlock()
	if !ok {
		t, err = e.genTemplate(views, helpers, file)
		if err != nil {
			return err
		}
	}

	return t.Execute(w, vars)

}

func (e *Engine) genTemplate(views *lazyview.Views, helpers map[string]any, file string) (*template.Template, error) {
	e.tplsL.Lock()
	defer e.tplsL.Unlock()
	f, err := views.FS.Open(file)
	if err != nil {
		e.tplsL.Unlock()
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	tpl := template.New(file)
	tpl = tpl.Funcs(helpers)
	switch e.MissingKey {
	case MissingKeyZero:
		tpl = tpl.Option("missingkey=zero")
	case MissingKeyError:
		tpl = tpl.Option("missingkey=error")
	case MissingKeyInvalid:
		tpl = tpl.Option("missingkey=invalid")
	}

	tpl, err = tpl.Parse(string(data))
	if err != nil {
		return nil, err
	}
	if e.tpls == nil {
		e.tpls = make(map[string](*template.Template))
	}
	e.tpls[file] = tpl
	return tpl, nil
}
