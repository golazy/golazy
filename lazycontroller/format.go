package lazycontroller

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"golazy.dev/lazyturbo"
)

// Format identifies the response representation selected for a request.
type Format string

const (
	HTML        Format = "html"
	JSON        Format = "json"
	TurboFrame  Format = "turbo_frame"
	TurboStream Format = "turbo_stream"
	PNG         Format = "png"
	JPEG        Format = "jpeg"
	GIF         Format = "gif"
	Image       Format = "image"
	SSE         Format = "sse"
)

type formatContextKey struct{}

var globalFormats = newFormatRegistry()

func init() {
	RegisterFormat(HTML, "text/html", "application/xhtml+xml")
	RegisterFormat(JSON, "application/json", "application/problem+json")
	RegisterFormat(TurboStream, lazyturbo.StreamMIME)
	RegisterFormat(PNG, "image/png")
	RegisterFormat(JPEG, "image/jpeg", "image/pjpeg")
	RegisterFormat(GIF, "image/gif")
	RegisterFormat(Image, "image/*")
	RegisterFormat(SSE, "text/event-stream")
	RegisterFormatSuffix(HTML, "html")
	RegisterFormatSuffix(JSON, "json")
	RegisterFormatSuffix(TurboStream, "turbo_stream")
	RegisterFormatSuffix(PNG, "png")
	RegisterFormatSuffix(JPEG, "jpg", "jpeg", "jpe", "pjpeg")
	RegisterFormatSuffix(GIF, "gif")
	RegisterFormatSuffix(SSE, "sse", "event_stream")
}

type newFormatOptions struct {
	name     string
	suffixes []string
}

// FormatOption configures a custom format created by NewFormat.
type FormatOption func(*newFormatOptions)

// As sets the symbolic format name returned by NewFormat.
func As(name string) FormatOption {
	return func(options *newFormatOptions) {
		options.name = name
	}
}

// Suffix registers one or more URL suffixes for a custom format.
func Suffix(suffixes ...string) FormatOption {
	return func(options *newFormatOptions) {
		options.suffixes = append(options.suffixes, suffixes...)
	}
}

// NewFormat registers a custom MIME type and returns its symbolic format.
func NewFormat(contentType string, options ...FormatOption) Format {
	mediaType, err := parseMediaType(contentType)
	if err != nil {
		panic(fmt.Sprintf("lazycontroller: invalid content type %q: %v", contentType, err))
	}
	config := newFormatOptions{
		name: deriveFormatName(mediaType),
	}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	format := Format(normalizeFormatName(config.name))
	if format == "" {
		panic("lazycontroller: format name is required")
	}
	RegisterFormat(format, mediaType)
	if len(config.suffixes) == 0 {
		RegisterFormatSuffix(format, string(format))
	} else {
		RegisterFormatSuffix(format, config.suffixes...)
	}
	return format
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

type Formats map[Format]func() error

type Responses = Formats

// Wants runs the response handler matching the request format.
func (b *Base) Wants(formats Formats) error {
	return b.respondTo(formats)
}

// Respond runs the response handler matching the request format.
func (b *Base) Respond(responses Responses) error {
	return b.respondTo(Formats(responses))
}

func (b *Base) respondTo(formats Formats) error {
	if len(formats) == 0 {
		return Error(http.StatusNotAcceptable, fmt.Errorf("no response formats are available"))
	}
	format, ok := b.negotiateFormat(formats)
	if !ok {
		return Error(http.StatusNotAcceptable, fmt.Errorf("format %q is not available", b.Format()))
	}
	if b.writer != nil {
		addVary(b.writer.Header(), "Accept")
		if format == TurboFrame {
			addVary(b.writer.Header(), "Turbo-Frame")
		}
	}
	if response := formats[format]; response != nil {
		previous := b.format
		b.format = format
		defer func() {
			b.format = previous
		}()
		return response()
	}
	return Error(http.StatusNotAcceptable, fmt.Errorf("format %q is not available", format))
}

func (b *Base) negotiateFormat(formats Formats) (Format, bool) {
	if b.request != nil && lazyturbo.IsFrameRequest(b.request) {
		if _, ok := formats[TurboFrame]; ok {
			return TurboFrame, true
		}
		if _, ok := formats[HTML]; ok {
			return HTML, true
		}
		return "", false
	}
	if b.request != nil {
		if format, ok := b.request.Context().Value(formatContextKey{}).(Format); ok && format != "" {
			return selectFormat(formats, format)
		}
		accept := b.request.Header.Get("Accept")
		preferences := acceptPreferences(accept)
		if len(preferences) > 0 {
			for _, preference := range preferences {
				if preference.mediaType == "*/*" {
					return firstAvailableFormat(formats)
				}
				for _, format := range formatsForMediaType(preference.mediaType) {
					if selected, ok := selectFormat(formats, format); ok {
						return selected, true
					}
				}
			}
			return "", false
		}
	}
	return selectFormat(formats, HTML)
}

func selectFormat(formats Formats, format Format) (Format, bool) {
	if _, ok := formats[format]; ok {
		return format, true
	}
	if imageFormat(format) {
		if _, ok := formats[Image]; ok {
			return Image, true
		}
	}
	return "", false
}

// Format returns the negotiated response format for the current request.
func (b *Base) Format() Format {
	if b.format != "" {
		return b.format
	}
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

func (b *Base) IsPNG() bool {
	return b.Is(PNG)
}

func (b *Base) IsJPEG() bool {
	return b.Is(JPEG)
}

func (b *Base) IsGIF() bool {
	return b.Is(GIF)
}

func (b *Base) IsImage() bool {
	return b.Is(Image)
}

func (b *Base) IsSSE() bool {
	return b.Is(SSE)
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
	formats := r.formatsForContentType(contentType)
	if len(formats) == 0 {
		return "", false
	}
	return formats[0], true
}

func (r *formatRegistry) formatsForContentType(contentType string) []Format {
	contentType = normalizeContentType(contentType)
	if contentType == "" {
		return nil
	}
	wildcard := wildcardContentType(contentType)
	r.mu.RLock()
	defer r.mu.RUnlock()
	var formats []Format
	if format, ok := r.contentTypes[contentType]; ok {
		formats = appendFormat(formats, format)
	}
	if wildcard != "" {
		if format, ok := r.contentTypes[wildcard]; ok {
			formats = appendFormat(formats, format)
		}
	}
	return formats
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
	for _, preference := range acceptPreferences(accept) {
		if preference.mediaType == "*/*" {
			return HTML, true
		}
		for _, format := range formatsForMediaType(preference.mediaType) {
			return format, true
		}
	}
	return "", false
}

type acceptPreference struct {
	mediaType string
	q         float64
	index     int
}

func acceptPreferences(accept string) []acceptPreference {
	var preferences []acceptPreference
	for item := range strings.SplitSeq(accept, ",") {
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
		if q <= 0 {
			continue
		}
		preferences = append(preferences, acceptPreference{
			mediaType: normalizeContentType(mediaType),
			q:         q,
			index:     len(preferences),
		})
	}
	sort.SliceStable(preferences, func(i, j int) bool {
		if preferences[i].q == preferences[j].q {
			return preferences[i].index < preferences[j].index
		}
		return preferences[i].q > preferences[j].q
	})
	return preferences
}

func formatsForMediaType(mediaType string) []Format {
	mediaType = normalizeContentType(mediaType)
	formats := globalFormats.formatsForContentType(mediaType)
	if mediaType == "image/*" {
		formats = appendFormat(formats, PNG)
		formats = appendFormat(formats, JPEG)
		formats = appendFormat(formats, GIF)
	}
	return formats
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

func parseMediaType(contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		return "", err
	}
	mediaType = strings.ToLower(mediaType)
	if !strings.Contains(mediaType, "/") {
		return "", fmt.Errorf("missing slash")
	}
	return mediaType, nil
}

func wildcardContentType(contentType string) string {
	mediaType, err := parseMediaType(contentType)
	if err != nil || strings.HasSuffix(mediaType, "/*") {
		return ""
	}
	mediaTypeType, _, ok := strings.Cut(mediaType, "/")
	if !ok || mediaTypeType == "" {
		return ""
	}
	return mediaTypeType + "/*"
}

func deriveFormatName(mediaType string) string {
	_, subtype, ok := strings.Cut(mediaType, "/")
	if !ok {
		return ""
	}
	return normalizeFormatName(subtype)
}

func normalizeFormatName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	var builder strings.Builder
	previousUnderscore := false
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func imageFormat(format Format) bool {
	switch format {
	case PNG, JPEG, GIF:
		return true
	default:
		return false
	}
}

func firstAvailableFormat(formats Formats) (Format, bool) {
	preferred := []Format{HTML, JSON, TurboStream, PNG, JPEG, GIF, Image, SSE}
	for _, format := range preferred {
		if _, ok := formats[format]; ok {
			return format, true
		}
	}
	var custom []string
	for format := range formats {
		custom = append(custom, string(format))
	}
	sort.Strings(custom)
	if len(custom) == 0 {
		return "", false
	}
	return Format(custom[0]), true
}

func appendFormat(formats []Format, format Format) []Format {
	if format == "" {
		return formats
	}
	if slices.Contains(formats, format) {
		return formats
	}
	return append(formats, format)
}
