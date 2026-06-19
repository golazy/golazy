package lazyturbo

import (
	"fmt"
	"html"
	"mime"
	"net/http"
	"strings"
	"unicode"

	"golazy.dev/lazyview"
)

const (
	frameHeader      = "Turbo-Frame"
	prefetchHeader   = "X-Sec-Purpose"
	secPurposeHeader = "Sec-Purpose"
	StreamMIME       = "text/vnd.turbo-stream.html"
	htmlContentType  = "text/html; charset=utf-8"
)

// FrameOption configures a rendered <turbo-frame> element.
type FrameOption struct {
	attribute frameAttribute
}

type frameAttribute struct {
	name    string
	value   string
	boolean bool
}

// FrameID returns the Turbo frame id requested by r.
func FrameID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get(frameHeader))
}

// IsFrameRequest reports whether r was issued for a Turbo Frame.
func IsFrameRequest(r *http.Request) bool {
	return FrameID(r) != ""
}

// IsPrefetch reports whether r is a Turbo link prefetch request.
func IsPrefetch(r *http.Request) bool {
	if r == nil {
		return false
	}
	return headerToken(r.Header.Get(prefetchHeader), "prefetch") ||
		headerToken(r.Header.Get(secPurposeHeader), "prefetch")
}

// AcceptsStream reports whether r advertises Turbo Stream response support.
func AcceptsStream(r *http.Request) bool {
	if r == nil {
		return false
	}
	for _, part := range strings.Split(r.Header.Get("Accept"), ",") {
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err == nil && strings.EqualFold(mediaType, StreamMIME) {
			return true
		}
	}
	return false
}

func headerToken(value string, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

// Helpers returns the template helpers provided by lazyturbo.
func Helpers() map[string]any {
	return map[string]any{
		"turbo_frame":               Frame,
		"turbo_src":                 Src,
		"turbo_loading":             Loading,
		"turbo_busy":                Busy,
		"turbo_disabled":            Disabled,
		"turbo_target":              Target,
		"turbo_complete":            Complete,
		"turbo_recurse":             Recurse,
		"turbo_autoscroll":          Autoscroll,
		"turbo_autoscroll_block":    AutoscrollBlock,
		"turbo_autoscroll_behavior": AutoscrollBehavior,
		"turbo_action":              Action,
		"turbo_refresh":             Refresh,
		"turbo_refresh_morph":       RefreshMorph,
	}
}

// Frame renders _<id>_frame.html.tpl and wraps the result in a <turbo-frame>.
func Frame(ctx *lazyview.Context, id string, data any, opts ...FrameOption) (lazyview.Fragment, error) {
	if err := ValidateFrameID(id); err != nil {
		return lazyview.Fragment{}, err
	}
	if ctx == nil {
		return lazyview.Fragment{}, fmt.Errorf("lazyturbo: render turbo frame %q: view context is missing", id)
	}
	if ctx.Views == nil {
		return lazyview.Fragment{}, fmt.Errorf("lazyturbo: render turbo frame %q: views are missing", id)
	}

	variables := copyVariables(ctx.Variables)
	if data == nil {
		data = ctx.Data
	}
	if locals, ok := data.(map[string]any); ok {
		variables = copyVariables(locals)
	}
	body, err := ctx.Views.RenderString(lazyview.Options{
		Context:    ctx.Context,
		Request:    ctx.Request,
		Variables:  variables,
		Data:       data,
		Helpers:    ctx.Helpers(),
		Route:      ctx.Route,
		Namespace:  ctx.Namespace,
		Controller: ctx.Controller,
		Partial:    framePartial(id),
		Format:     "html",
		UseLayout:  false,
	})
	if err != nil {
		return lazyview.Fragment{}, err
	}
	return FrameTag(id, body, opts...)
}

// FrameTag wraps body in a <turbo-frame> tag.
func FrameTag(id string, body string, opts ...FrameOption) (lazyview.Fragment, error) {
	if err := ValidateFrameID(id); err != nil {
		return lazyview.Fragment{}, err
	}
	id = strings.TrimSpace(id)

	attributes := []frameAttribute{{name: "id", value: id}}
	for _, opt := range opts {
		attr := opt.attribute
		if attr.name == "" {
			continue
		}
		if err := validateFrameAttribute(attr); err != nil {
			return lazyview.Fragment{}, err
		}
		attributes = append(attributes, attr)
	}

	var builder strings.Builder
	builder.WriteString("<turbo-frame")
	for _, attr := range attributes {
		builder.WriteByte(' ')
		builder.WriteString(attr.name)
		if !attr.boolean {
			builder.WriteString(`="`)
			builder.WriteString(html.EscapeString(attr.value))
			builder.WriteByte('"')
		}
	}
	builder.WriteByte('>')
	builder.WriteString(body)
	builder.WriteString("</turbo-frame>")
	return lazyview.Fragment{
		Body:        builder.String(),
		ContentType: htmlContentType,
	}, nil
}

func framePartial(id string) string {
	return strings.TrimSpace(id) + "_frame"
}

// ValidateFrameID checks whether id is safe to use as both a DOM id and a
// frame partial name.
func ValidateFrameID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("lazyturbo: frame id is required")
	}
	for _, r := range id {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '_', '-', ':':
			continue
		default:
			return fmt.Errorf("lazyturbo: frame id %q contains invalid character %q", id, r)
		}
	}
	return nil
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

// Src sets the frame src attribute.
func Src(src string) FrameOption {
	return frameOption("src", src)
}

// Loading sets the frame loading attribute. Valid values are eager and lazy.
func Loading(loading string) FrameOption {
	return frameOption("loading", loading)
}

// Busy sets the frame busy boolean attribute.
func Busy() FrameOption {
	return boolFrameOption("busy")
}

// Disabled sets the frame disabled boolean attribute.
func Disabled() FrameOption {
	return boolFrameOption("disabled")
}

// Target sets the frame target attribute.
func Target(target string) FrameOption {
	return frameOption("target", target)
}

// Complete sets the frame complete boolean attribute.
func Complete() FrameOption {
	return boolFrameOption("complete")
}

// Recurse sets the frame recurse attribute.
func Recurse(frameID string) FrameOption {
	return frameOption("recurse", frameID)
}

// Autoscroll sets the frame autoscroll boolean attribute.
func Autoscroll() FrameOption {
	return boolFrameOption("autoscroll")
}

// AutoscrollBlock sets data-autoscroll-block. Valid values are end, start, center, and nearest.
func AutoscrollBlock(block string) FrameOption {
	return frameOption("data-autoscroll-block", block)
}

// AutoscrollBehavior sets data-autoscroll-behavior. Valid values are auto and smooth.
func AutoscrollBehavior(behavior string) FrameOption {
	return frameOption("data-autoscroll-behavior", behavior)
}

// Action sets data-turbo-action. Valid values are advance and replace.
func Action(action string) FrameOption {
	return frameOption("data-turbo-action", action)
}

// Refresh sets the frame refresh attribute. Turbo currently defines morph.
func Refresh(refresh string) FrameOption {
	return frameOption("refresh", refresh)
}

// RefreshMorph sets refresh="morph".
func RefreshMorph() FrameOption {
	return Refresh("morph")
}

func frameOption(name string, value string) FrameOption {
	return FrameOption{attribute: frameAttribute{name: name, value: strings.TrimSpace(value)}}
}

func boolFrameOption(name string) FrameOption {
	return FrameOption{attribute: frameAttribute{name: name, boolean: true}}
}

func validateFrameAttribute(attr frameAttribute) error {
	if attr.boolean {
		return nil
	}
	if attr.value == "" {
		return fmt.Errorf("lazyturbo: %s value is required", attr.name)
	}
	switch attr.name {
	case "loading":
		return validateOneOf(attr.name, attr.value, "eager", "lazy")
	case "data-autoscroll-block":
		return validateOneOf(attr.name, attr.value, "end", "start", "center", "nearest")
	case "data-autoscroll-behavior":
		return validateOneOf(attr.name, attr.value, "auto", "smooth")
	case "data-turbo-action":
		return validateOneOf(attr.name, attr.value, "advance", "replace")
	case "refresh":
		return validateOneOf(attr.name, attr.value, "morph")
	default:
		return nil
	}
}

func validateOneOf(name string, value string, allowed ...string) error {
	for _, candidate := range allowed {
		if value == candidate {
			return nil
		}
	}
	return fmt.Errorf("lazyturbo: invalid %s value %q", name, value)
}
