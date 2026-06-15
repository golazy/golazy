package lazycontroller

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"golazy.dev/lazyview"
)

// Renderer is the application view renderer.
type Renderer = lazyview.Views

type rendererContextKey struct{}
type writerContextKey struct{}
type requestContextKey struct{}
type routeContextKey struct{}

func NewRenderer(views fs.FS) (*Renderer, error) {
	return lazyview.New(views)
}

func WithRenderer(ctx context.Context, renderer *Renderer) context.Context {
	return context.WithValue(ctx, rendererContextKey{}, renderer)
}

func WithWriter(ctx context.Context, writer http.ResponseWriter) context.Context {
	return context.WithValue(ctx, writerContextKey{}, writer)
}

func WithRequest(ctx context.Context, request *http.Request) context.Context {
	return context.WithValue(ctx, requestContextKey{}, request)
}

func WithRoute(ctx context.Context, route lazyview.Route) context.Context {
	return context.WithValue(ctx, routeContextKey{}, route)
}

type Base struct {
	ctx        context.Context
	request    *http.Request
	writer     http.ResponseWriter
	renderer   *Renderer
	route      lazyview.Route
	controller string
	layout     string
	data       map[string]any
	helpers    map[string]any
}

func NewBase(ctx context.Context, viewPath ...string) (Base, error) {
	renderer, ok := ctx.Value(rendererContextKey{}).(*Renderer)
	if !ok {
		return Base{}, fmt.Errorf("renderer is missing from application context")
	}
	writer, ok := ctx.Value(writerContextKey{}).(http.ResponseWriter)
	if !ok {
		return Base{}, fmt.Errorf("response writer is missing from controller context")
	}
	request, _ := ctx.Value(requestContextKey{}).(*http.Request)
	route, _ := ctx.Value(routeContextKey{}).(lazyview.Route)

	controller := route.Controller
	if len(viewPath) > 0 && strings.TrimSpace(viewPath[0]) != "" {
		controller = viewPath[0]
	}
	if controller == "" {
		return Base{}, fmt.Errorf("controller route metadata is missing from controller context")
	}

	return Base{
		ctx:        ctx,
		request:    request,
		writer:     writer,
		renderer:   renderer,
		route:      route,
		controller: controller,
		layout:     "app",
		data:       make(map[string]any),
		helpers:    make(map[string]any),
	}, nil
}

func (b *Base) Set(name string, value any) {
	b.data[name] = value
}

func (b *Base) SetLayout(layout string) {
	b.layout = layout
}

func (b *Base) Helper(name string, helper any) {
	if b.helpers == nil {
		b.helpers = make(map[string]any)
	}
	b.helpers[name] = helper
}

func (b *Base) Helpers(helpers map[string]any) {
	for name, helper := range helpers {
		b.Helper(name, helper)
	}
}

func (b *Base) Render(view string) error {
	if b.writer == nil || b.renderer == nil {
		return fmt.Errorf("controller base is not initialized")
	}
	if view == "" {
		view = strings.ToLower(b.route.Action)
	}
	if view == "" {
		return fmt.Errorf("view name is required")
	}

	return b.renderer.Render(lazyview.Options{
		Context:    b.ctx,
		Request:    b.request,
		Writer:     b.writer,
		Variables:  b.data,
		Helpers:    b.helpers,
		Route:      b.route,
		Namespace:  b.route.Namespace,
		Controller: b.controller,
		Action:     view,
		Layout:     b.layout,
		UseLayout:  true,
	})
}
