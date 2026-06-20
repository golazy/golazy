package lazycontroller

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"golazy.dev/lazyturbo"
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
	useLayout  bool
	status     int
	data       map[string]any
	helpers    map[string]any
	frameOpts  []lazyturbo.FrameOption
	format     Format
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
		useLayout:  true,
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
	b.useLayout = true
	b.status = 0
	b.data = make(map[string]any)
	b.helpers = make(map[string]any)
	b.frameOpts = nil
	return nil
}

func (b *Base) ResetRequest() {
	b.ctx = b.appCtx
	b.request = nil
	b.writer = nil
	b.route = lazyview.Route{}
	b.controller = b.viewPath
	b.useLayout = true
	b.status = 0
	b.data = nil
	b.helpers = nil
	b.frameOpts = nil
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

func (b *Base) Layout(layout string) {
	b.layout = layout
	b.useLayout = true
}

func (b *Base) SetLayout(layout string) {
	b.Layout(layout)
}

func (b *Base) NoLayout() {
	b.useLayout = false
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
	return b.render(view, b.Format())
}

func (b *Base) RenderHTML(view string) error {
	return b.render(view, HTML)
}

func (b *Base) render(view string, format Format) error {
	if b.writer == nil || b.renderer == nil {
		return fmt.Errorf("controller base is not initialized")
	}
	if view == "" {
		view = strings.ToLower(b.route.Action)
	}
	if view == "" {
		return fmt.Errorf("view name is required")
	}

	addVary(b.writer.Header(), "Accept")
	if b.IsTurboFrame() {
		return b.renderTurboFrame(lazyturbo.FrameID(b.request), b.frameOpts)
	}

	return b.renderView(lazyview.Options{
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
		UseLayout:  b.useLayout,
	}, format)
}

func (b *Base) SetTurboFrameOptions(opts ...lazyturbo.FrameOption) {
	b.frameOpts = append([]lazyturbo.FrameOption(nil), opts...)
}

func (b *Base) RenderTurboFrame(id string, opts ...lazyturbo.FrameOption) error {
	if b.writer == nil || b.renderer == nil {
		return fmt.Errorf("controller base is not initialized")
	}
	return b.renderTurboFrame(id, opts)
}

func (b *Base) renderTurboFrame(id string, opts []lazyturbo.FrameOption) error {
	id = strings.TrimSpace(id)
	if err := lazyturbo.ValidateFrameID(id); err != nil {
		return err
	}
	addVary(b.writer.Header(), "Accept", "Turbo-Frame")

	body, err := b.renderer.RenderString(lazyview.Options{
		Context:    b.ctx,
		Request:    b.request,
		Variables:  b.data,
		Helpers:    b.helpers,
		Route:      b.route,
		Namespace:  b.route.Namespace,
		Controller: b.controller,
		Partial:    id + "_frame",
		Format:     string(HTML),
		UseLayout:  false,
	})
	if err != nil {
		return err
	}
	frame, err := lazyturbo.FrameTag(id, body, opts...)
	if err != nil {
		return err
	}
	if b.writer.Header().Get("Content-Type") == "" {
		b.writer.Header().Set("Content-Type", frame.ContentType)
	}
	buffer := newBufferedResponse()
	copyHeaders(buffer.Header(), b.writer.Header())
	_, _ = buffer.Write([]byte(frame.Body))
	return b.writeBufferedResponse(buffer)
}

func (b *Base) renderView(options lazyview.Options, format Format) error {
	options.Format = string(format)
	if b.IsTurboFrame() {
		options.Format = string(HTML)
	}
	if options.Format != string(HTML) {
		options.UseLayout = false
	}

	buffer := newBufferedResponse()
	copyHeaders(buffer.Header(), b.writer.Header())
	options.Writer = buffer
	if err := b.renderer.Render(options); err != nil {
		return err
	}
	return b.writeBufferedResponse(buffer)
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
	status int
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
	w.status = status
	w.sent = true
}

func (b *Base) writeBufferedResponse(buffer *bufferedResponse) error {
	if buffer == nil {
		return fmt.Errorf("controller response buffer is missing")
	}
	status := buffer.status
	if b.status != 0 {
		status = b.status
	}
	if status == 0 {
		status = http.StatusOK
	}
	if status < 100 || status > 999 {
		return fmt.Errorf("lazycontroller: invalid response status %d", status)
	}

	copyHeaders(b.writer.Header(), buffer.Header())
	b.writer.WriteHeader(status)
	if buffer.body.Len() == 0 {
		return nil
	}
	_, err := b.writer.Write(buffer.body.Bytes())
	return err
}

func copyHeaders(target http.Header, source http.Header) {
	for key, values := range source {
		target.Del(key)
		target[key] = append([]string(nil), values...)
	}
}

func addVary(header http.Header, values ...string) {
	if header == nil {
		return
	}
	existing := map[string]bool{}
	for _, value := range header.Values("Vary") {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				existing[strings.ToLower(part)] = true
			}
		}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || existing[strings.ToLower(value)] {
			continue
		}
		header.Add("Vary", value)
		existing[strings.ToLower(value)] = true
	}
}
