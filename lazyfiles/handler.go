package lazyfiles

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Handler serves fallback application file URLs.
func (f *Files) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	prefix := strings.TrimRight(f.RoutePrefix, "/")
	if prefix == "" {
		prefix = "/_lazy/files"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.URL.Path, prefix+"/")
		if token == r.URL.Path || token == "" || strings.Contains(token, "/") {
			next.ServeHTTP(w, r)
			return
		}
		f.serveToken(w, r, token)
	})
}

func (f *Files) serveToken(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	id, err := f.verifyToken(token)
	if err != nil {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	opened, file, _, err := f.Open(r.Context(), id)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, fmt.Errorf("open file: %w", err).Error(), http.StatusInternalServerError)
		return
	}
	defer opened.Close()
	if file.ContentType != "" {
		w.Header().Set("Content-Type", file.ContentType)
	}
	if file.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))
	}
	if file.Checksum != "" {
		w.Header().Set("ETag", strconv.Quote(file.Checksum))
	}
	if r.Method == http.MethodHead {
		return
	}
	if seeker, ok := opened.(io.ReadSeeker); ok {
		http.ServeContent(w, r, file.Filename, file.UpdatedAt, seeker)
		return
	}
	_, _ = io.Copy(w, opened)
}
