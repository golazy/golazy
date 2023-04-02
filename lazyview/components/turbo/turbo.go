package turbo

import (
	"golazy.dev/lazyview/component"
	"golazy.dev/lazyview/nodes"
	"golazy.dev/lazyview/script"
)

var Component = component.Register(&component.Npm{
	Name:    "@hotwired/turbo",
	Imports: component.ImportMap{"@hotwired/turbo": "dist/turbo.es2017-esm.js"},
	Scripts: []script.Script{
		{
			Content: `import * as Turbo from "@hotwired/turbo"; Turbo.start(); `,
			Data: map[string]string{
				"turbo-track": "reload",
			},
		},
	},
})

var ReloadAttr = map[string]string{"turbo-track": "reload"}

func TurboFrame(options ...any) nodes.Element {
	return nodes.NewElement("turbo-frame", options...)
}

func Src(value ...string) nodes.Attr {
	return nodes.NewAttr("src", value...)
}
func Loading(value ...string) nodes.Attr {
	return nodes.NewAttr("loading", value...)
}
func Busy(value ...string) nodes.Attr {
	return nodes.NewAttr("budy", value...)
}
func Disabled(value ...string) nodes.Attr {
	return nodes.NewAttr("disabled", value...)
}
func Targets(value ...string) nodes.Attr {
	return nodes.NewAttr("targets", value...)
}
func Target(value ...string) nodes.Attr {
	return nodes.NewAttr("target", value...)
}
func Complete(value ...string) nodes.Attr {
	return nodes.NewAttr("complete", value...)
}
func Autoscroll(value ...string) nodes.Attr {
	return nodes.NewAttr("autoscroll", value...)
}
func Action(value ...string) nodes.Attr {
	return nodes.NewAttr("action", value...)
}

func TurboStream(options ...any) nodes.Element {
	return nodes.NewElement("turbo-stream", options...)
}

func DataTurbo(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo", value...)
}
func DataTurboTrack(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-track", value...)
}
func DataTurboFrame(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-frame ", value...)
}
func DataTurboAction(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-action ", value...)
}
func DataTurboPermanent(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-permanent ", value...)
}
func DataTurboCache(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-cache", value...)
}
func DataTurboEval(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-eval", value...)
}
func DataTurboMethod(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-method ", value...)
}
func DataTurboStream(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-stream ", value...)
}
func DataTurboConfirm(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-confirm ", value...)
}
func DataTurboPreload(value ...string) nodes.Attr {
	return nodes.NewAttr("data-turbo-preload", value...)
}
