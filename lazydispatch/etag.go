package lazydispatch

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
)

// ETag adds conditional response handling for buffered dynamic responses.
func ETag() Middleware {
	return etagMiddleware{}
}

type etagMiddleware struct{}

func (etagMiddleware) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buffer, ok := w.(*BufferedResponseWriter)
		if ok {
			next.ServeHTTP(buffer, r)
			applyETag(buffer, r)
			return
		}

		buffer = NewBufferedResponseWriter(w)
		next.ServeHTTP(buffer, r)
		applyETag(buffer, r)
		_ = buffer.Flush()
	})
}

func applyETag(w *BufferedResponseWriter, r *http.Request) {
	if !etagEligible(w, r) {
		return
	}

	etag := w.Header().Get("ETag")
	if etag == "" {
		etag = etagFor(w.body.Bytes())
		w.Header().Set("ETag", etag)
	}

	if !etagMatches(r.Header.Get("If-None-Match"), etag) {
		return
	}

	headers := cloneNotModifiedHeaders(w.Header())
	w.Reset()
	for key, values := range headers {
		w.Header()[key] = values
	}
	w.WriteHeader(http.StatusNotModified)
}

func etagEligible(w *BufferedResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if !w.sent && len(w.header) == 0 {
		return false
	}
	if w.status != 0 && w.status != http.StatusOK {
		return false
	}
	if strings.Contains(strings.ToLower(w.Header().Get("Cache-Control")), "no-store") {
		return false
	}
	if w.Header().Get("Content-Encoding") != "" {
		return false
	}
	if len(w.Header().Values("Set-Cookie")) != 0 {
		return false
	}
	return true
}

func etagFor(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%q", fmt.Sprintf("%x", sum[:]))
}

func etagMatches(header, etag string) bool {
	header = strings.TrimSpace(header)
	if header == "" || etag == "" {
		return false
	}
	if header == "*" {
		return true
	}
	for _, candidate := range splitETagList(header) {
		if weakETagValue(candidate) == weakETagValue(etag) {
			return true
		}
	}
	return false
}

func weakETagValue(etag string) string {
	etag = strings.TrimSpace(etag)
	if strings.HasPrefix(etag, "W/") || strings.HasPrefix(etag, "w/") {
		etag = strings.TrimSpace(etag[2:])
	}
	return etag
}

func splitETagList(header string) []string {
	var values []string
	start := 0
	inQuote := false
	escaped := false
	for index, char := range header {
		switch {
		case escaped:
			escaped = false
		case char == '\\' && inQuote:
			escaped = true
		case char == '"':
			inQuote = !inQuote
		case char == ',' && !inQuote:
			values = append(values, strings.TrimSpace(header[start:index]))
			start = index + 1
		}
	}
	values = append(values, strings.TrimSpace(header[start:]))
	return values
}

func cloneNotModifiedHeaders(headers http.Header) http.Header {
	keep := http.Header{}
	for _, key := range []string{
		"Cache-Control",
		"Content-Location",
		"Date",
		"ETag",
		"Expires",
		"Last-Modified",
		"Vary",
	} {
		values := headers.Values(key)
		if len(values) == 0 {
			continue
		}
		keep[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
	}
	return keep
}
