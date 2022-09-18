package main

import (
	"context"
	_ "embed"
	"io"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	. "github.com/golazy/golazy/lazyview/html"
	"github.com/golazy/golazy/lazyview/layout"
	"github.com/golazy/golazy/lazyview/nodes"
)

var (
	cli *client.Client
)

func init() {
	d, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli = d
}

type Controller struct {
}

//go:embed style.css
var style string

var PageLayout = layout.LayoutTemplate{
	Lang:     "en",
	Title:    "golazy",
	Viewport: "width=device-width",
	Head:     []interface{}{Script(Type("module"), Src("https://cdn.skypack.dev/@hotwired/turbo"))},
	Styles:   []string{style},
	LayoutBody: func(l *layout.LayoutTemplate, content ...interface{}) io.WriterTo {
		return nodes.ContentNode{
			Header(
				H1("webdocker"),
			),
			Nav(
				"Homepage",
			),
			Main(
				content,
			),
		}
	},
}

func (c *Controller) Layout(r *http.Request) *layout.LayoutTemplate {
	return &PageLayout
}

func (c *Controller) Index(w http.ResponseWriter, r *http.Request) interface{} {
	var rows []interface{}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, container := range containers {
		url := "/container/" + container.ID

		rows = append(rows, Tr(
			Td(container.ID[:10]),
			Td(container.Image),
			Td(
				A(Href(url), "Inspect"),
				Form(Method("delete"), Action(url), Input(Type("submit"), "Stop")),
			),
		))
	}

	return Table(
		Thead(
			Tr(
				Td("containers"),
				Td("id"),
				Td("actions"),
			),
		),
		Tbody(rows),
	)
}
