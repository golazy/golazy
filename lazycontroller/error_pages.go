package lazycontroller

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
)

type errorPagesContextKey struct{}

func WithErrorPages(ctx context.Context, files fs.FS) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if files == nil {
		return ctx
	}
	return context.WithValue(ctx, errorPagesContextKey{}, files)
}

func WriteErrorPage(ctx context.Context, w http.ResponseWriter, r *http.Request, status int) bool {
	if status < 400 || status > 599 {
		return false
	}
	return WriteFile(ctx, w, r, strconv.Itoa(status)+".html", status) == nil
}

func WriteErrorFallback(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	return WriteFile(ctx, w, r, "500.html", http.StatusInternalServerError) == nil
}

func WriteFile(ctx context.Context, w http.ResponseWriter, r *http.Request, file string, status int) error {
	if ctx == nil || w == nil {
		return fmt.Errorf("lazycontroller: response context is missing")
	}
	if status == 0 {
		status = http.StatusOK
	}
	if status < 100 || status > 999 {
		return fmt.Errorf("lazycontroller: invalid response status %d", status)
	}
	files, ok := ctx.Value(errorPagesContextKey{}).(fs.FS)
	if !ok || files == nil {
		return fmt.Errorf("lazycontroller: public file system is missing")
	}

	file = path.Clean("/" + file)[1:]
	if !fs.ValidPath(file) {
		return fmt.Errorf("lazycontroller: invalid public file %q", file)
	}
	body, err := fs.ReadFile(files, file)
	if err != nil {
		return fmt.Errorf("lazycontroller: read public file %q: %w", file, err)
	}

	ResetResponse(w)
	contentType := mime.TypeByExtension(path.Ext(file))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if r != nil && r.Method == http.MethodHead {
		return nil
	}
	_, err = w.Write(body)
	return err
}
