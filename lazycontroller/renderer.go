package lazycontroller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golazy.dev/lazycache"
	"golazy.dev/lazysession"
	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
)

const maxPooledControllerBufferBody = 64 << 10

var controllerResponseBufferPool = sync.Pool{
	New: func() any {
		return &bufferedResponse{}
	},
}

// Renderer is the application view renderer.
type Renderer = lazyview.Views

type rendererContextKey struct{}

func NewRenderer(views fs.FS) (*Renderer, error) {
	return lazyview.New(views)
}

func WithRenderer(ctx context.Context, renderer *Renderer) context.Context {
	return context.WithValue(ctx, rendererContextKey{}, renderer)
}

func RendererFromContext(ctx context.Context) (*Renderer, bool) {
	renderer, ok := ctx.Value(rendererContextKey{}).(*Renderer)
	return renderer, ok
}

type Base struct {
	appCtx       context.Context
	ctx          context.Context
	request      *http.Request
	writer       http.ResponseWriter
	renderer     *Renderer
	route        lazyview.Route
	viewPath     string
	controller   string
	layout       string
	useLayout    bool
	status       int
	data         map[string]any
	helpers      map[string]any
	frameOpts    []lazyturbo.FrameOption
	format       Format
	variants     []string
	cacheKey     cacheKeySpec
	session      *lazysession.Session
	sessionSet   bool
	sessionDirty bool
}

type cacheKeySpec struct {
	set   bool
	full  bool
	parts []any
	key   string
	hit   bool
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

	requestContext := r.Context()
	hasAppContext := b.hasAppContext(requestContext)
	if !hasAppContext {
		requestContext = withAppContext(requestContext, b.appCtx)
	}
	b.ctx = requestContext
	if hasAppContext {
		b.request = r
	} else {
		b.request = r.WithContext(requestContext)
	}
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
	b.variants = nil
	b.cacheKey = cacheKeySpec{}
	b.session = nil
	b.sessionSet = false
	b.sessionDirty = false
	return nil
}

func (b *Base) hasAppContext(ctx context.Context) bool {
	renderer, ok := RendererFromContext(ctx)
	return ok && renderer == b.renderer
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
	b.variants = nil
	b.cacheKey = cacheKeySpec{}
	b.session = nil
	b.sessionSet = false
	b.sessionDirty = false
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

func (b *Base) ResponseWriter() http.ResponseWriter {
	return b.writer
}

func (b *Base) Cache() *lazycache.Cache {
	if b == nil {
		return nil
	}
	cache, _ := lazycache.FromContext(b.ctx)
	return cache
}

func (b *Base) CacheKey(parts ...any) bool {
	return b.setCacheKey(false, parts...)
}

func (b *Base) CacheKeyF(parts ...any) bool {
	return b.setCacheKey(true, parts...)
}

func (b *Base) setCacheKey(full bool, parts ...any) bool {
	b.cacheKey = cacheKeySpec{set: true, full: full, parts: append([]any(nil), parts...)}
	key, ok, err := b.renderCacheKey(strings.ToLower(b.route.Action), b.Format())
	if err != nil || !ok {
		return false
	}
	b.cacheKey.key = key
	cache := b.Cache()
	if cache == nil {
		return false
	}
	body, err := lazycache.Get[string](cache, key)
	if err != nil {
		return false
	}
	buffer := newBufferedResponse()
	copyHeaders(buffer.Header(), b.writer.Header())
	format := b.Format()
	if b.IsTurboFrame() {
		format = HTML
	}
	if buffer.Header().Get("Content-Type") == "" {
		buffer.Header().Set("Content-Type", contentTypeForFormat(format))
	}
	_, _ = buffer.Write([]byte(body))
	if b.writeBufferedResponse(buffer) != nil {
		return false
	}
	b.cacheKey.hit = true
	return true
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

func (b *Base) RenderSVGString(view string, variants ...string) (string, error) {
	return b.renderString(view, "svg", variants...)
}

func (b *Base) renderString(view string, format string, variants ...string) (string, error) {
	if b.renderer == nil {
		return "", fmt.Errorf("controller base is not initialized")
	}
	if view == "" {
		view = strings.ToLower(b.route.Action)
	}
	if view == "" {
		return "", fmt.Errorf("view name is required")
	}
	return b.renderer.RenderString(lazyview.Options{
		Context:    b.ctx,
		Request:    b.request,
		Variables:  b.data,
		Helpers:    b.helpers,
		Route:      b.route,
		Namespace:  b.route.Namespace,
		Controller: b.controller,
		Action:     view,
		Format:     format,
		Variants:   variants,
		UseLayout:  false,
	})
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
		return b.renderTurboFrame(lazyturbo.FrameID(b.request), b.frameOpts, view)
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
		Variants:   b.variants,
		Layout:     b.layout,
		UseLayout:  b.useLayout,
	}, format)
}

func (b *Base) Variants(variants ...string) {
	b.variants = append([]string(nil), variants...)
}

func (b *Base) SetTurboFrameOptions(opts ...lazyturbo.FrameOption) {
	b.frameOpts = append([]lazyturbo.FrameOption(nil), opts...)
}

func (b *Base) RenderTurboFrame(id string, opts ...lazyturbo.FrameOption) error {
	if b.writer == nil || b.renderer == nil {
		return fmt.Errorf("controller base is not initialized")
	}
	return b.renderTurboFrame(id, opts, strings.TrimSpace(id)+"_frame")
}

func (b *Base) renderTurboFrame(id string, opts []lazyturbo.FrameOption, action string) error {
	if b.cacheKey.hit {
		return nil
	}
	id = strings.TrimSpace(id)
	if err := lazyturbo.ValidateFrameID(id); err != nil {
		return err
	}
	addVary(b.writer.Header(), "Accept", "Turbo-Frame")
	if key, ok, err := b.renderCacheKey(action, b.Format()); err != nil {
		return err
	} else if ok {
		return b.renderCachedTurboFrame(id, opts, key)
	}

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
		Variants:   b.variants,
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

func (b *Base) renderCachedTurboFrame(id string, opts []lazyturbo.FrameOption, key string) error {
	cache := b.Cache()
	if cache == nil {
		return fmt.Errorf("lazycontroller: cache is missing from application context")
	}
	if body, err := lazycache.Get[string](cache, key); err == nil {
		buffer := newBufferedResponse()
		copyHeaders(buffer.Header(), b.writer.Header())
		if buffer.Header().Get("Content-Type") == "" {
			buffer.Header().Set("Content-Type", contentTypeForFormat(HTML))
		}
		_, _ = buffer.Write([]byte(body))
		return b.writeBufferedResponse(buffer)
	} else if err != nil && !errors.Is(err, lazycache.ErrMiss) {
		return err
	}

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
		Variants:   b.variants,
		UseLayout:  false,
	})
	if err != nil {
		return err
	}
	frame, err := lazyturbo.FrameTag(id, body, opts...)
	if err != nil {
		return err
	}
	if err := cache.Set(frame.Body, key); err != nil {
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
	if b.cacheKey.hit {
		return nil
	}
	options.Format = string(format)
	if b.IsTurboFrame() {
		options.Format = string(HTML)
	}
	if options.Format != string(HTML) {
		options.UseLayout = false
	}
	if key, ok, err := b.renderCacheKey(options.Action, format); err != nil {
		return err
	} else if ok {
		return b.renderCachedView(options, format, key)
	}

	buffer := newBufferedResponse()
	copyHeaders(buffer.Header(), b.writer.Header())
	options.Writer = buffer
	if err := b.renderer.Render(options); err != nil {
		releaseBufferedResponse(buffer)
		return err
	}
	return b.writeBufferedResponse(buffer)
}

func (b *Base) renderCachedView(options lazyview.Options, format Format, key string) error {
	cache := b.Cache()
	if cache == nil {
		return fmt.Errorf("lazycontroller: cache is missing from application context")
	}
	if body, err := lazycache.Get[string](cache, key); err == nil {
		buffer := newBufferedResponse()
		copyHeaders(buffer.Header(), b.writer.Header())
		if buffer.Header().Get("Content-Type") == "" {
			buffer.Header().Set("Content-Type", contentTypeForFormat(format))
		}
		_, _ = buffer.Write([]byte(body))
		return b.writeBufferedResponse(buffer)
	} else if err != nil && !errors.Is(err, lazycache.ErrMiss) {
		return err
	}

	buffer := newBufferedResponse()
	copyHeaders(buffer.Header(), b.writer.Header())
	options.Writer = buffer
	if err := b.renderer.Render(options); err != nil {
		releaseBufferedResponse(buffer)
		return err
	}
	if err := cache.Set(buffer.body.String(), key); err != nil {
		releaseBufferedResponse(buffer)
		return err
	}
	return b.writeBufferedResponse(buffer)
}

func contentTypeForFormat(format Format) string {
	switch format {
	case JSON:
		return "application/json; charset=utf-8"
	case TurboStream:
		return "text/vnd.turbo-stream.html; charset=utf-8"
	default:
		return "text/html; charset=utf-8"
	}
}

func (b *Base) renderCacheKey(action string, format Format) (string, bool, error) {
	if !b.cacheKey.set {
		return "", false, nil
	}
	if b.cacheKey.key != "" {
		return b.cacheKey.key, true, nil
	}
	scope := b.cacheKeyScopeParts()
	if b.cacheKey.full {
		parts := append(scope, b.cacheKey.parts...)
		key, err := lazycache.Key(parts...)
		return key, true, err
	}
	parts := scope
	if strings.TrimSpace(b.route.Namespace) != "" {
		parts = append(parts, b.route.Namespace)
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = strings.ToLower(b.route.Action)
	}
	parts = append(parts, b.controller, action, string(format))
	parts = append(parts, b.cacheKey.parts...)
	key, err := lazycache.Key(parts...)
	return key, true, err
}

func (b *Base) cacheKeyScopeParts() []any {
	parts := []any{"build", lazycache.BuildVersionFromContext(b.ctx)}
	var variants []string
	for _, variant := range b.variants {
		if variant = strings.TrimSpace(variant); variant != "" {
			variants = append(variants, variant)
		}
	}
	if len(variants) > 0 {
		parts = append(parts, "variant", strings.Join(variants, "+"))
	}
	return parts
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

	detailErrors := DetailErrors(b.ctx)
	status := StatusCode(err)
	b.Set("status", status)
	b.Set("statusText", http.StatusText(status))
	b.Set("detailErrors", detailErrors)
	if detailErrors {
		b.Set("error", err.Error())
		b.Set("backtrace", errorBacktrace(err))
		b.Set("openEditorPath", openEditorPath(b.ctx))
	} else {
		b.Set("error", "")
		b.Set("backtrace", []errorFrame(nil))
		b.Set("openEditorPath", "")
	}

	if b.renderer == nil {
		if b.returnFile(w, r, strconv.Itoa(status)+".html", status) == nil {
			return nil
		}
		return fmt.Errorf("controller renderer is not initialized")
	}
	if w == nil {
		return fmt.Errorf("response writer is missing")
	}

	buffer := newBufferedResponse()
	defer releaseBufferedResponse(buffer)
	buffer.WriteHeader(status)
	renderErr := b.renderer.Render(lazyview.Options{
		Context:    b.ctx,
		Request:    r,
		Writer:     buffer,
		Variables:  b.data,
		Helpers:    b.helpers,
		Route:      b.route,
		Controller: "app",
		Action:     "error",
		Layout:     b.layout,
		UseLayout:  true,
	})
	if renderErr != nil {
		if b.returnFile(w, r, strconv.Itoa(status)+".html", status) == nil {
			return nil
		}
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
	buffer := controllerResponseBufferPool.Get().(*bufferedResponse)
	buffer.init()
	return buffer
}

func releaseBufferedResponse(w *bufferedResponse) {
	if w == nil {
		return
	}
	if w.header != nil {
		clear(w.header)
	}
	w.status = 0
	w.sent = false
	if w.body.Cap() > maxPooledControllerBufferBody {
		return
	}
	w.body.Reset()
	controllerResponseBufferPool.Put(w)
}

func (w *bufferedResponse) init() {
	if w.header == nil {
		w.header = make(http.Header)
	} else {
		clear(w.header)
	}
	w.body.Reset()
	w.status = 0
	w.sent = false
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
	defer releaseBufferedResponse(buffer)
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
		for part := range strings.SplitSeq(value, ",") {
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
