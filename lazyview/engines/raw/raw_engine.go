package raw

import (
	"context"
	"io"

	"golazy.dev/lazyview"
)

var (
	Extensions = []string{
		"txt",
		"html",
		"htm",
		"xml",
		"json",
		"yaml",
		"js",
	}
)

type Engine struct {
}

var _ lazyview.Engine = &Engine{}

func (e *Engine) Render(ctx context.Context, views *lazyview.Views, w io.Writer, variables map[string]any, helpers map[string]any, file string) error {
	f, err := views.FS.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	return err

}
