package lazycontroller

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path"
)

type Renderer struct {
	views fs.FS
}

type rendererContextKey struct{}
type writerContextKey struct{}

func NewRenderer(views fs.FS) (*Renderer, error) {
	if views == nil {
		return nil, fmt.Errorf("views filesystem is required")
	}
	if _, err := fs.Stat(views, "layouts/app.html.tpl"); err != nil {
		return nil, fmt.Errorf("load default layout: %w", err)
	}
	return &Renderer{views: views}, nil
}

func WithRenderer(ctx context.Context, renderer *Renderer) context.Context {
	return context.WithValue(ctx, rendererContextKey{}, renderer)
}

func WithWriter(ctx context.Context, writer http.ResponseWriter) context.Context {
	return context.WithValue(ctx, writerContextKey{}, writer)
}

type Base struct {
	writer   http.ResponseWriter
	renderer *Renderer
	viewPath string
	layout   string
	data     map[string]any
}

func NewBase(ctx context.Context, viewPath string) (Base, error) {
	renderer, ok := ctx.Value(rendererContextKey{}).(*Renderer)
	if !ok {
		return Base{}, fmt.Errorf("renderer is missing from application context")
	}
	writer, ok := ctx.Value(writerContextKey{}).(http.ResponseWriter)
	if !ok {
		return Base{}, fmt.Errorf("response writer is missing from controller context")
	}

	return Base{
		writer:   writer,
		renderer: renderer,
		viewPath: viewPath,
		layout:   "app",
		data:     make(map[string]any),
	}, nil
}

func (b *Base) Set(name string, value any) {
	b.data[name] = value
}

func (b *Base) SetLayout(layout string) {
	b.layout = layout
}

func (b *Base) Render(view string) error {
	if b.writer == nil || b.renderer == nil {
		return fmt.Errorf("controller base is not initialized")
	}

	viewFile := path.Join(b.viewPath, view+".html.tpl")
	viewTemplate, err := template.ParseFS(b.renderer.views, viewFile)
	if err != nil {
		return fmt.Errorf("parse view %q: %w", viewFile, err)
	}

	var content bytes.Buffer
	if err := viewTemplate.Execute(&content, b.data); err != nil {
		return fmt.Errorf("execute view %q: %w", viewFile, err)
	}

	layoutFile := path.Join("layouts", b.layout+".html.tpl")
	layoutTemplate, err := template.ParseFS(b.renderer.views, layoutFile)
	if err != nil {
		return fmt.Errorf("parse layout %q: %w", layoutFile, err)
	}

	layoutData := make(map[string]any, len(b.data)+1)
	for name, value := range b.data {
		layoutData[name] = value
	}
	layoutData["content"] = template.HTML(content.String())

	var page bytes.Buffer
	if err := layoutTemplate.Execute(&page, layoutData); err != nil {
		return fmt.Errorf("execute layout %q: %w", layoutFile, err)
	}

	b.writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	b.writer.WriteHeader(http.StatusOK)
	if _, err := page.WriteTo(b.writer); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}
