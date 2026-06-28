package middlewares

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"golazy.dev/lazydispatch"
)

const defaultMethodOverrideScan = 64 * 1024

type originalMethodKey struct{}

func MethodOverride() lazydispatch.Middleware {
	return MethodOverrideWithLimit(defaultMethodOverrideScan)
}

func MethodOverrideWithLimit(maxScan int64) lazydispatch.Middleware {
	return methodOverride{maxScan: maxScan}
}

type methodOverride struct {
	maxScan int64
}

func (methodOverride) MiddlewareName() string {
	return "lazydispatch.middlewares.MethodOverride"
}

func (middleware methodOverride) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldSkipMethodOverride(r) {
			next.ServeHTTP(w, r)
			return
		}

		method, present, valid := readMethodOverride(r, middleware.maxScan)
		if present && !valid {
			http.Error(w, "invalid _method", http.StatusBadRequest)
			return
		}
		if valid {
			r = r.WithContext(context.WithValue(r.Context(), originalMethodKey{}, r.Method))
			r.Method = strings.ToUpper(method)
		}
		next.ServeHTTP(w, r)
	})
}

func OriginalMethod(r *http.Request) string {
	if r == nil {
		return ""
	}
	if method, ok := r.Context().Value(originalMethodKey{}).(string); ok {
		return method
	}
	return ""
}

func shouldSkipMethodOverride(r *http.Request) bool {
	if r == nil || r.Body == nil || r.Method != http.MethodPost {
		return true
	}
	if isUpgradeRequest(r) {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return true
	}
	return mediaType != "application/x-www-form-urlencoded" && mediaType != "multipart/form-data"
}

func readMethodOverride(r *http.Request, maxScan int64) (string, bool, bool) {
	prefix, body := readPrefix(r.Body, maxScan)
	r.Body = replayReadCloser(prefix, body)

	mediaType, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var method string
	switch mediaType {
	case "application/x-www-form-urlencoded":
		method = readURLEncodedMethod(prefix)
	case "multipart/form-data":
		method = readMultipartMethod(prefix, params["boundary"])
	}
	if method == "" {
		return "", false, false
	}
	method = strings.ToLower(method)
	switch method {
	case "put", "patch", "delete":
		return method, true, true
	default:
		return method, true, false
	}
}

func readPrefix(body io.ReadCloser, maxScan int64) ([]byte, io.ReadCloser) {
	if maxScan < 0 {
		maxScan = 0
	}
	var buffer bytes.Buffer
	_, _ = io.Copy(&buffer, io.LimitReader(body, maxScan))
	return buffer.Bytes(), body
}

func readURLEncodedMethod(prefix []byte) string {
	values, err := url.ParseQuery(string(prefix))
	if err != nil {
		return ""
	}
	return values.Get("_method")
}

func readMultipartMethod(prefix []byte, boundary string) string {
	if boundary == "" {
		return ""
	}
	reader := multipart.NewReader(bytes.NewReader(prefix), boundary)
	for {
		part, err := reader.NextPart()
		if err != nil {
			return ""
		}
		if part.FileName() != "" {
			return ""
		}
		if part.FormName() != "_method" {
			continue
		}
		value, _ := io.ReadAll(io.LimitReader(part, 32))
		return string(value)
	}
}

func isUpgradeRequest(r *http.Request) bool {
	if r.Header.Get("Upgrade") != "" {
		return true
	}
	for _, value := range r.Header.Values("Connection") {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "upgrade") {
				return true
			}
		}
	}
	return false
}

func replayReadCloser(prefix []byte, body io.ReadCloser) io.ReadCloser {
	return &replayBody{
		Reader: io.MultiReader(bytes.NewReader(prefix), body),
		Closer: body,
	}
}

type replayBody struct {
	io.Reader
	io.Closer
}
