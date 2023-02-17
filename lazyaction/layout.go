package lazyaction

import "io"

type Layout interface {
	Render(Context, io.Writer) error
}

type OptimizedLayout interface {
	OptimizedRender(Context, io.Writer) error
}
