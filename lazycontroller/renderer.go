package lazycontroller

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"golazy.dev/lazyview"
)

// Renderer is the application view renderer.
type Renderer = lazyview.Views

type rendererContextKey struct{}

func NewRenderer(views fs.FS) (*Renderer, error) {
	return lazyview.New(views)
}

func WithRenderer(ctx context.Context, renderer *Renderer) context.Context {
	return context.WithValue(ctx, rendererContextKey{}, renderer)
}

type Base struct {
	appCtx     context.Context
	ctx        context.Context
	request    *http.Request
	writer     http.ResponseWriter
	renderer   *Renderer
	route      lazyview.Route
	viewPath   string
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
	controller := ""
	if len(viewPath) > 0 && strings.TrimSpace(viewPath[0]) != "" {
		controller = strings.TrimSpace(viewPath[0])
	}

	return Base{
		appCtx:     ctx,
		ctx:        ctx,
		renderer:   renderer,
		viewPath:   controller,
		controller: controller,
		layout:     "app",
	}, nil
}

func (b *Base) BindRequest(w http.ResponseWriter, r *http.Request, route lazyview.Route) error {
	if w == nil {
		return fmt.Errorf("response writer is missing from controller request")
	}
	if r == nil {
		return fmt.Errorf("request is missing from controller request")
	}
	if b.renderer == nil {
		return fmt.Errorf("controller renderer is not initialized")
	}

	controller := b.viewPath
	if controller == "" {
		controller = route.Controller
	}
	if controller == "" {
		return fmt.Errorf("controller route metadata is missing from controller request")
	}

	requestContext := withAppContext(r.Context(), b.appCtx)
	b.ctx = requestContext
	b.request = r.WithContext(requestContext)
	b.writer = w
	b.route = route
	b.controller = controller
	if b.layout == "" {
		b.layout = "app"
	}
	b.data = make(map[string]any)
	b.helpers = make(map[string]any)
	return nil
}

func (b *Base) ResetRequest() {
	b.ctx = b.appCtx
	b.request = nil
	b.writer = nil
	b.route = lazyview.Route{}
	b.controller = b.viewPath
	b.data = nil
	b.helpers = nil
}

type appContext struct {
	context.Context
	app context.Context
}

func withAppContext(ctx context.Context, app context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if app == nil {
		return ctx
	}
	return appContext{Context: ctx, app: app}
}

func (c appContext) Value(key any) any {
	if value := c.Context.Value(key); value != nil {
		return value
	}
	return c.app.Value(key)
}

func (b *Base) Set(name string, value any) {
	if b.data == nil {
		b.data = make(map[string]any)
	}
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

func (b *Base) Request() *http.Request {
	return b.request
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

func (b *Base) HandleError(w http.ResponseWriter, r *http.Request, err error) error {
	if err == nil {
		return nil
	}
	if w == nil {
		w = b.writer
	}
	if r == nil {
		r = b.request
	}
	ResetResponse(w)

	status := StatusCode(err)
	b.Set("status", status)
	b.Set("statusText", http.StatusText(status))
	b.Set("error", err.Error())

	if b.returnFile(w, r, strconv.Itoa(status)+".html", status) == nil {
		return nil
	}
	if b.renderer == nil {
		return fmt.Errorf("controller renderer is not initialized")
	}
	if w == nil {
		return fmt.Errorf("response writer is missing")
	}

	buffer := newBufferedResponse()
	buffer.WriteHeader(status)
	renderErr := b.renderer.Render(lazyview.Options{
		Context:    b.ctx,
		Request:    r,
		Writer:     buffer,
		Variables:  b.data,
		Helpers:    b.helpers,
		Route:      b.route,
		Namespace:  b.route.Namespace,
		Controller: b.controller,
		Action:     "error",
		Layout:     b.layout,
		UseLayout:  true,
	})
	if renderErr != nil {
		return renderErr
	}

	ResetResponse(w)
	copyHeaders(w.Header(), buffer.Header())
	w.WriteHeader(status)
	_, writeErr := w.Write(buffer.body.Bytes())
	return writeErr
}

func (b *Base) ReturnFile(file string, status int) error {
	return b.returnFile(nil, nil, file, status)
}

func (b *Base) ServeErrorPage(w http.ResponseWriter, r *http.Request, status int) bool {
	return b.returnFile(w, r, strconv.Itoa(status)+".html", status) == nil
}

func (b *Base) returnFile(w http.ResponseWriter, r *http.Request, file string, status int) error {
	if w == nil {
		w = b.writer
	}
	if r == nil {
		r = b.request
	}
	return WriteFile(b.ctx, w, r, file, status)
}

type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	sent   bool
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{
		header: make(http.Header),
	}
}

func (w *bufferedResponse) Header() http.Header {
	return w.header
}

func (w *bufferedResponse) Write(data []byte) (int, error) {
	if !w.sent {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(data)
}

func (w *bufferedResponse) WriteHeader(status int) {
	if w.sent {
		return
	}
	w.sent = true
}

func copyHeaders(target http.Header, source http.Header) {
	for key, values := range source {
		target.Del(key)
		target[key] = append([]string(nil), values...)
	}
}
