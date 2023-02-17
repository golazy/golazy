package layout_manager

import "io"

func RegisterLayout(l Layout, am AssetManager) {
	am.RegisterAsset(l)
	return l
}

type Headerer interface {
	Header() []any
}

var BasicLayout = &Layout{
	Lang:  "en",
	Title: "golazy",
	Components: []Headerer{
		EsModuleShims,
		Turbo,
	},
}

type Layouter interface {
	Header() *Header
	Render(io.WriterTo) io.WriterTo
}

type Titler interface {
	Title() string
}
type Langer interface {
	Lang() string
}

type LayoutParenting interface {
	ParentLayout() Layouter
}

func Render(l Layouter, content io.WriterTo) io.WriterTo {

	h := &Header{}
	// Merge headers from layout and parent layouts

	cl := l
	for cl != nil {
		h.Merge(cl.Header())
		content = cl.Render(content)

		if pl, ok := cl.(LayoutParenting); ok {
			cl = pl.ParentLayout()
		} else {
			break
		}
	}

	return nil
}
