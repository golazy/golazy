package lazyview

import (
	"context"
	"io"
)

type Engine interface {
	Render(ctx context.Context, views *Views, w io.Writer, vars map[string]any, helpers map[string]any, file string) error
}
