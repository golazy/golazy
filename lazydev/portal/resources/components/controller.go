package components

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"portal/layouts/golazy"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyview/component"
	. "golazy.dev/lazyview/html"
)

type Controller struct {
	golazy.Layout
}

func (c *Controller) GenComponent(id string) component.Component {
	panic(id)
	return component.Find(id)
}

func Each[T any](items []T, fn func(T) any) []any {

	data := make([]any, len(items))
	for i, item := range items {
		data[i] = fn(item)
	}

	return data
}

func (c *Controller) Index(ctx *lazyaction.Context) io.WriterTo {

	return Div(
		Code(Pre(fmt.Sprint(component.DefaultInstallOptions))),
		Table(
			Thead(
				Tr(Th("ID"), Th("Installed"), Th("")),
			),
			Tbody(
				Each(component.All(), func(c component.ComponentState) any {
					return Tr(
						Td(c.Name),
						Td(c.Installed),
						Td(A("Install", Href("/golazy/components/"+url.PathEscape(c.Name)+"/install"))),
					)
				}),
			),
		),
	)

}

func (c *Controller) MemberGetInstall(r *http.Request, cmp component.Component, id string) io.WriterTo {
	return Dl(
		Dt("Name"),
		Dd(),
	)
}
