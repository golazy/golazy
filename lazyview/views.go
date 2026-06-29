package lazyview

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"golazy.dev/lazytelemetry"
	"golazy.dev/lazytelemetry/lazytracing"
)

const maxPooledRenderBufferBody = 64 << 10

var renderBufferPool = sync.Pool{
	New: func() any {
		return &bytes.Buffer{}
	},
}

// Views owns the application view filesystem, registered engines, and global helpers.
type Views struct {
	FS      fs.FS
	Engines map[string]Engine
	Helpers map[string]any
}

// Options configures one render operation.
type Options struct {
	Context context.Context
	Request *http.Request
	Writer  io.Writer

	Variables map[string]any
	// Data overrides the value used as dot while executing the template.
	Data    any
	Helpers map[string]any
	Route   Route

	Namespace  string
	Controller string
	Action     string
	Partial    string
	Format     string
	Variants   []string
	Layout     string
	UseLayout  bool
}

// New builds a view set using the engines registered by imported engine packages.
func New(files fs.FS) (*Views, error) {
	if files == nil {
		return nil, fmt.Errorf("lazyview: views filesystem is required")
	}
	if _, err := fs.Stat(files, "layouts/app.html.tpl"); err != nil {
		return nil, fmt.Errorf("lazyview: load default layout: %w", err)
	}
	views := &Views{
		FS:      files,
		Engines: registeredEngines(),
		Helpers: map[string]any{},
	}
	views.AddHelpers(views.baseHelpers())
	return views, nil
}

// Cache builds or rebuilds template caches for engines that support caching.
//
// Applications should register helpers before calling Cache. If helpers are
// changed later, AddHelpers clears existing caches and Cache should be called
// again before serving requests.
func (v *Views) Cache() error {
	if v == nil {
		return fmt.Errorf("lazyview: views is nil")
	}
	helpers := copyHelpers(v.Helpers)
	for extension, engine := range v.Engines {
		cacheable, ok := engine.(CacheableEngine)
		if !ok {
			continue
		}
		if err := cacheable.Cache(CacheContext{
			FS:        v.FS,
			Extension: extension,
			Helpers:   helpers,
		}); err != nil {
			return fmt.Errorf("lazyview: cache %s templates: %w", extension, err)
		}
	}
	return nil
}

func (v *Views) clearCache() {
	for _, engine := range v.Engines {
		if clearer, ok := engine.(CacheClearer); ok {
			clearer.ClearCache()
		}
	}
}

// Render renders a view and optionally wraps it with a layout.
func (v *Views) Render(options Options) error {
	if v == nil {
		return fmt.Errorf("lazyview: views is nil")
	}
	if options.Writer == nil {
		return fmt.Errorf("lazyview: writer is required")
	}
	renderContext := v.renderContext(options)

	file, err := v.findView(renderContext)
	if err != nil {
		return err
	}

	if !options.UseLayout {
		setContentType(options.Writer, renderContext.Format)
		return v.renderNamedTemplate(renderContext, options.Writer, file, "view.render")
	}

	content := acquireRenderBuffer()
	defer releaseRenderBuffer(content)
	if err := v.renderNamedTemplate(renderContext, content, file, "view.render"); err != nil {
		return err
	}

	layoutFile, err := v.findLayout(renderContext)
	if err != nil {
		return err
	}
	layoutContext := *renderContext
	layoutContext.Action = ""
	layoutContext.Partial = ""
	layoutContext.Variables = copyVariables(renderContext.Variables)
	layoutContext.Variables["content"] = Fragment{
		Body:        content.String(),
		ContentType: contentTypeForFormat(renderContext.Format),
	}
	layoutContext.Data = layoutContext.Variables

	setContentType(options.Writer, renderContext.Format)
	return v.renderNamedTemplate(&layoutContext, options.Writer, layoutFile, "view.layout")
}

// RenderString renders a view to a string.
func (v *Views) RenderString(options Options) (string, error) {
	out := acquireRenderBuffer()
	defer releaseRenderBuffer(out)
	options.Writer = out
	if err := v.Render(options); err != nil {
		return "", err
	}
	return out.String(), nil
}

func acquireRenderBuffer() *bytes.Buffer {
	buffer := renderBufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	return buffer
}

func releaseRenderBuffer(buffer *bytes.Buffer) {
	if buffer == nil {
		return
	}
	if buffer.Cap() > maxPooledRenderBufferBody {
		return
	}
	buffer.Reset()
	renderBufferPool.Put(buffer)
}

func (v *Views) renderContext(options Options) *Context {
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	format := options.Format
	if format == "" {
		format = "html"
	}
	layout := options.Layout
	if layout == "" {
		layout = "app"
	}
	variables := copyVariables(options.Variables)
	data := options.Data
	if data == nil {
		data = variables
	}
	helpers := make(map[string]any, len(v.Helpers)+len(options.Helpers))
	for name, helper := range v.Helpers {
		helpers[name] = helper
	}
	for name, helper := range options.Helpers {
		helpers[name] = helper
	}

	return &Context{
		Context:    ctx,
		Request:    options.Request,
		Views:      v,
		Route:      options.Route,
		Variables:  variables,
		Data:       data,
		helpers:    helpers,
		Namespace:  firstNonEmpty(options.Namespace, options.Route.Namespace),
		Controller: firstNonEmpty(options.Controller, options.Route.Controller),
		Action:     options.Action,
		Partial:    options.Partial,
		Format:     format,
		Variants:   normalizeVariants(options.Variants),
		Layout:     layout,
	}
}

func (v *Views) renderTemplate(ctx *Context, writer io.Writer, file string) error {
	extension := strings.TrimPrefix(filepath.Ext(file), ".")
	engine, ok := v.Engines[extension]
	if !ok {
		return fmt.Errorf("lazyview: no engine registered for %q", extension)
	}
	return engine.Render(ctx, writer, file)
}

func (v *Views) renderNamedTemplate(ctx *Context, writer io.Writer, file string, regionKind string) (err error) {
	if ctx == nil {
		return v.renderTemplate(ctx, writer, file)
	}
	if lazytelemetry.SpanFromContext(ctx.Context) == nil {
		return v.renderTemplate(ctx, writer, file)
	}
	regionCtx, span := lazytelemetry.StartRegion(ctx.Context, templateRegionName(regionKind, file), renderAttributes(ctx, file)...)
	if span == nil {
		return v.renderTemplate(ctx, writer, file)
	}
	defer span.End()
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	renderContext := *ctx
	renderContext.Context = regionCtx
	return v.renderTemplate(&renderContext, writer, file)
}

func renderAttributes(ctx *Context, file string) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("view.file", file),
		slog.String("view.format", ctx.Format),
		slog.String("view.controller", ctx.Controller),
		slog.String("view.action", ctx.Action),
	}
	if ctx.Partial != "" {
		attrs = append(attrs, slog.String("view.partial", ctx.Partial))
	}
	if ctx.Layout != "" {
		attrs = append(attrs, slog.String("view.layout", ctx.Layout))
	}
	if ctx.Route.Name != "" {
		attrs = append(attrs, slog.String("route.name", ctx.Route.Name))
	}
	return attrs
}

func templateRegionName(kind string, file string) string {
	if file == "" {
		return kind
	}
	return kind + " " + file
}

func (v *Views) findView(ctx *Context) (string, error) {
	name := ctx.Action
	if ctx.Partial != "" {
		name = "_" + ctx.Partial
	}
	if name == "" {
		return "", fmt.Errorf("lazyview: action or partial is required")
	}

	var tried []string
	for _, directory := range viewDirectories(ctx) {
		file, ok := v.findFile(directory, name, ctx.Format, ctx.Variants, &tried)
		if ok {
			return file, nil
		}
	}
	return "", fmt.Errorf("lazyview: view not found. Tried: %s", strings.Join(tried, ", "))
}

func viewDirectories(ctx *Context) []string {
	var directories []string
	if ctx.Controller != "" {
		controller := strings.Trim(ctx.Controller, "/")
		namespace := strings.Trim(ctx.Namespace, "/")
		if namespace != "" && !strings.HasPrefix(controller, namespace+"/") {
			controller = path.Join(namespace, controller)
		}
		directories = append(directories, controller)
	} else if ctx.Namespace != "" {
		directories = append(directories, strings.Trim(ctx.Namespace, "/"))
	}
	return append(directories, "app")
}

func (v *Views) findLayout(ctx *Context) (string, error) {
	layouts := []string{}
	if ctx.Layout != "" {
		layouts = append(layouts, ctx.Layout)
	}
	if ctx.Controller != "" {
		layouts = append(layouts, ctx.Controller)
	}
	layouts = append(layouts, "app")

	directories := []string{"layouts"}
	if ctx.Namespace != "" {
		directories = append([]string{path.Join("layouts", ctx.Namespace)}, directories...)
	}

	var tried []string
	for _, directory := range directories {
		for _, layout := range layouts {
			file, ok := v.findFile(directory, layout, ctx.Format, ctx.Variants, &tried)
			if ok {
				return file, nil
			}
		}
	}
	return "", fmt.Errorf("lazyview: layout not found. Tried: %s", strings.Join(tried, ", "))
}

func (v *Views) findFile(directory string, name string, format string, variants []string, tried *[]string) (string, bool) {
	for extension := range v.Engines {
		for _, variant := range variants {
			candidate := path.Join(directory, name+"."+format+"+"+variant+"."+extension)
			*tried = append(*tried, candidate)
			info, err := fs.Stat(v.FS, candidate)
			if err == nil && !info.IsDir() {
				return candidate, true
			}
		}
		candidate := path.Join(directory, name+"."+format+"."+extension)
		*tried = append(*tried, candidate)
		info, err := fs.Stat(v.FS, candidate)
		if err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return "", false
}

func (ctx *Context) partial(args ...any) (Fragment, error) {
	if len(args) == 0 {
		return Fragment{}, fmt.Errorf("lazyview: partial requires a name")
	}
	name, ok := args[0].(string)
	if !ok || strings.TrimSpace(name) == "" {
		return Fragment{}, fmt.Errorf("lazyview: partial name must be a string")
	}

	variables := copyVariables(ctx.Variables)
	data := ctx.Data
	if data == nil {
		data = variables
	}
	if len(args) > 1 {
		if len(args) > 2 {
			return Fragment{}, fmt.Errorf("lazyview: partial expects at most 2 arguments")
		}
		data = args[1]
		if locals, ok := data.(map[string]any); ok {
			variables = copyVariables(locals)
		}
	}

	partialCtx := ctx.Context
	var span *lazytracing.Span
	if lazytelemetry.SpanFromContext(ctx.Context) != nil {
		partialCtx, span = lazytelemetry.StartRegion(ctx.Context, partialRegionName(name),
			slog.String("view.partial", name),
			slog.String("view.controller", ctx.Controller),
			slog.String("view.format", ctx.Format),
		)
	}
	if span != nil {
		defer span.End()
	}
	body, err := ctx.Views.RenderString(Options{
		Context:    partialCtx,
		Request:    ctx.Request,
		Variables:  variables,
		Data:       data,
		Helpers:    ctx.helpers,
		Route:      ctx.Route,
		Namespace:  ctx.Namespace,
		Controller: ctx.Controller,
		Partial:    name,
		Format:     ctx.Format,
		Variants:   ctx.Variants,
		UseLayout:  false,
	})
	if err != nil {
		if span != nil {
			span.RecordError(err)
		}
		return Fragment{}, err
	}
	return Fragment{
		Body:        body,
		ContentType: contentTypeForFormat(ctx.Format),
	}, nil
}

func partialRegionName(name string) string {
	if name == "" {
		return "view.partial"
	}
	return "view.partial " + name
}

func copyVariables(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	for name, value := range source {
		out[name] = value
	}
	return out
}

func setContentType(writer io.Writer, format string) {
	response, ok := writer.(http.ResponseWriter)
	if !ok {
		return
	}
	if response.Header().Get("Content-Type") == "" {
		response.Header().Set("Content-Type", contentTypeForFormat(format))
	}
}

func contentTypeForFormat(format string) string {
	switch format {
	case "json":
		return "application/json; charset=utf-8"
	case "svg":
		return "image/svg+xml; charset=utf-8"
	case "turbo_stream":
		return "text/vnd.turbo-stream.html; charset=utf-8"
	default:
		return "text/html; charset=utf-8"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeVariants(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	var variants []string
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.ContainsAny(value, `/\.`) || seen[value] {
			continue
		}
		seen[value] = true
		variants = append(variants, value)
	}
	return variants
}
