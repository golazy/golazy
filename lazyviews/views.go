package lazyviews

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ViewsPath is the local view directory used by lazydev builds.
var ViewsPath = "app/views"

// Open returns a filesystem rooted at ViewsPath, searching from the current
// working directory up through its parents when ViewsPath is relative.
func Open() (fs.FS, error) {
	dir, err := Resolve(ViewsPath)
	if err != nil {
		return nil, err
	}
	return os.DirFS(dir), nil
}

// Resolve returns the concrete local directory for a development view path.
func Resolve(viewPath string) (string, error) {
	viewPath = strings.TrimSpace(viewPath)
	if viewPath == "" {
		return "", fmt.Errorf("lazyviews: views path is required")
	}

	if filepath.IsAbs(viewPath) {
		if viewsDirExists(viewPath) {
			return viewPath, nil
		}
		return "", missingViewsError(viewPath)
	}

	if viewsDirExists(viewPath) {
		return viewPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("lazyviews: get working directory: %w", err)
	}
	for {
		candidate := filepath.Join(cwd, viewPath)
		if viewsDirExists(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return "", missingViewsError(viewPath)
}

func viewsDirExists(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "layouts", "app.html.tpl"))
	return err == nil && !info.IsDir()
}

func missingViewsError(viewPath string) error {
	return fmt.Errorf(
		"lazyviews: views path %q does not contain layouts/app.html.tpl",
		viewPath,
	)
}
