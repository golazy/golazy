package lazycontroller

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"golazy.dev/lazyturbo"
)

// Format identifies the response representation selected for a request.
type Format string

const (
	HTML        Format = "html"
	JSON        Format = "json"
	TurboFrame  Format = "turbo_frame"
	TurboStream Format = "turbo_stream"
)

type formatContextKey struct{}

var globalFormats = newFormatRegistry()

func init() {
	RegisterFormat(HTML, "text/html", "application/xhtml+xml")
	RegisterFormat(JSON, "application/json")
	RegisterFormat(TurboStream, lazyturbo.StreamMIME)
	RegisterFormatSuffix(HTML, "html")
	RegisterFormatSuffix(JSON, "json")
	RegisterFormatSuffix(TurboStream, "turbo_stream")
}

// RegisterFormat maps one or more content types to a controller format.
func RegisterFormat(format Format, contentTypes ...string) {
	globalFormats.registerContentTypes(format, contentTypes...)
}

// RegisterFormatSuffix maps one or more URL suffixes to a controller format.
func RegisterFormatSuffix(format Format, suffixes ...string) {
	globalFormats.registerSuffixes(format, suffixes...)
}

// FormatFromContentType resolves a content type to a registered format.
func FormatFromContentType(contentType string) (Format, bool) {
	return globalFormats.formatForContentType(contentType)
}

// FormatFromSuffix resolves a URL suffix to a registered format.
func FormatFromSuffix(suffix string) (Format, bool) {
	return globalFormats.formatForSuffix(suffix)
}

// WithFormat returns a context that carries an explicitly requested format.
func WithFormat(ctx context.Context, format Format) context.Context {
	return context.WithValue(ctx, formatContextKey{}, format)
}

// FormatFromRequest returns the request format inferred from Turbo headers,
// explicit route metadata, and Accept.
func FormatFromRequest(r *http.Request) Format {
	if lazyturbo.IsFrameRequest(r) {
		return TurboFrame
	}
	if r != nil {
		if format, ok := r.Context().Value(formatContextKey{}).(Format); ok && format != "" {
			return format
		}
		if format, ok := formatFromAccept(r.Header.Get("Accept")); ok {
			return format
		}
	}
	return HTML
}

type Responses map[Format]func() error

// Respond runs the response handler matching the request format.
func (b *Base) Respond(responses Responses) error {
	if len(responses) == 0 {
		return Error(http.StatusNotAcceptable, fmt.Errorf("no response formats are available"))
	}
	format := b.Format()
	if b.writer != nil {
		addVary(b.writer.Header(), "Accept")
		if format == TurboFrame {
			addVary(b.writer.Header(), "Turbo-Frame")
		}
	}
	if response := responses[format]; response != nil {
		return response()
	}
	if format == TurboFrame {
		if response := responses[HTML]; response != nil {
			return response()
		}
	}
	return Error(http.StatusNotAcceptable, fmt.Errorf("format %q is not available", format))
}

// Format returns the negotiated response format for the current request.
func (b *Base) Format() Format {
	return FormatFromRequest(b.request)
}

// Is reports whether the current request resolved to format.
func (b *Base) Is(format Format) bool {
	return b.Format() == format
}

func (b *Base) IsHTML() bool {
	return b.Is(HTML)
}

func (b *Base) IsJSON() bool {
	return b.Is(JSON)
}

func (b *Base) IsTurboFrame() bool {
	return b.Is(TurboFrame)
}

func (b *Base) IsTurboStream() bool {
	return b.Is(TurboStream)
}

type formatRegistry struct {
	mu           sync.RWMutex
	contentTypes map[string]Format
	suffixes     map[string]Format
}

func newFormatRegistry() *formatRegistry {
	return &formatRegistry{
		contentTypes: map[string]Format{},
		suffixes:     map[string]Format{},
	}
}

func (r *formatRegistry) registerContentTypes(format Format, contentTypes ...string) {
	if format == "" {
		panic("lazycontroller: format is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, contentType := range contentTypes {
		contentType = normalizeContentType(contentType)
		if contentType == "" {
			panic("lazycontroller: content type is required")
		}
		r.contentTypes[contentType] = format
	}
}

func (r *formatRegistry) registerSuffixes(format Format, suffixes ...string) {
	if format == "" {
		panic("lazycontroller: format is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, suffix := range suffixes {
		suffix = normalizeSuffix(suffix)
		if suffix == "" {
			panic("lazycontroller: suffix is required")
		}
		r.suffixes[suffix] = format
	}
}

func (r *formatRegistry) formatForContentType(contentType string) (Format, bool) {
	contentType = normalizeContentType(contentType)
	if contentType == "" {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	format, ok := r.contentTypes[contentType]
	return format, ok
}

func (r *formatRegistry) formatForSuffix(suffix string) (Format, bool) {
	suffix = normalizeSuffix(suffix)
	if suffix == "" {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	format, ok := r.suffixes[suffix]
	return format, ok
}

func formatFromAccept(accept string) (Format, bool) {
	var selected Format
	selectedQ := -1.0
	for _, item := range strings.Split(accept, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		mediaType, params, err := mime.ParseMediaType(item)
		if err != nil {
			continue
		}
		q := 1.0
		if rawQ := params["q"]; rawQ != "" {
			parsed, err := strconv.ParseFloat(rawQ, 64)
			if err != nil {
				continue
			}
			q = parsed
		}
		if q <= 0 || q <= selectedQ {
			continue
		}
		if format, ok := FormatFromContentType(mediaType); ok {
			selected = format
			selectedQ = q
			continue
		}
		if mediaType == "*/*" {
			selected = HTML
			selectedQ = q
		}
	}
	if selected == "" {
		return "", false
	}
	return selected, true
}

func normalizeContentType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		contentType = mediaType
	}
	return strings.ToLower(contentType)
}

func normalizeSuffix(suffix string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(suffix)), ".")
}
