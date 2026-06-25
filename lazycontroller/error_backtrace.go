package lazycontroller

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golazy.dev/lazyerrors"
)

type openEditorPathContextKey struct{}

// WithOpenEditorPath records the development endpoint used by error pages to
// open a backtrace frame in the user's editor.
func WithOpenEditorPath(ctx context.Context, path string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, openEditorPathContextKey{}, strings.TrimSpace(path))
}

func openEditorPath(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	path, _ := ctx.Value(openEditorPathContextKey{}).(string)
	return path
}

type errorFrame struct {
	Function     string
	File         string
	AbsoluteFile string
	Line         int
}

func (f errorFrame) String() string {
	switch {
	case f.Function == "" && f.File == "":
		return ""
	case f.Function == "":
		return formatFileLine(f.File, f.Line)
	case f.File == "":
		return f.Function
	default:
		return fmt.Sprintf("%s %s", f.Function, formatFileLine(f.File, f.Line))
	}
}

func errorBacktrace(err error) []errorFrame {
	var traced interface {
		Backtrace() []lazyerrors.Frame
	}
	if !errors.As(err, &traced) {
		return nil
	}

	formatter := newErrorPathFormatter()
	frames := traced.Backtrace()
	backtrace := make([]errorFrame, 0, len(frames))
	for _, frame := range frames {
		absoluteFile := ""
		if frame.File != "" {
			absoluteFile = filepath.Clean(frame.File)
		}
		backtrace = append(backtrace, errorFrame{
			Function:     frame.Function,
			File:         formatter.displayFile(absoluteFile),
			AbsoluteFile: absoluteFile,
			Line:         frame.Line,
		})
	}
	return backtrace
}

type errorPathFormatter struct {
	roots []string
}

func newErrorPathFormatter() errorPathFormatter {
	roots := make([]string, 0, 2)
	workingDirectory, err := os.Getwd()
	if err == nil {
		if root := goWorkRoot(workingDirectory); root != "" {
			roots = append(roots, root)
		}
		roots = append(roots, workingDirectory)
	}
	return errorPathFormatter{roots: cleanUniqueRoots(roots)}
}

func (f errorPathFormatter) displayFile(file string) string {
	if file == "" {
		return ""
	}
	file = filepath.Clean(file)
	for _, root := range f.roots {
		if relative, ok := relativeToRoot(root, file); ok {
			return filepath.ToSlash(relative)
		}
	}
	if relative, ok := relativeToModuleCache(file); ok {
		return relative
	}
	if relative, ok := relativeToModule(file); ok {
		return relative
	}
	return filepath.ToSlash(file)
}

func cleanUniqueRoots(roots []string) []string {
	cleaned := make([]string, 0, len(roots))
	seen := map[string]struct{}{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		cleaned = append(cleaned, root)
	}
	return cleaned
}

func goWorkRoot(workingDirectory string) string {
	if gowork := strings.TrimSpace(os.Getenv("GOWORK")); gowork != "" && gowork != "off" {
		if filepath.Base(gowork) == "go.work" {
			return filepath.Dir(gowork)
		}
		return gowork
	}

	for dir := filepath.Clean(workingDirectory); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
	}
}

func relativeToRoot(root string, file string) (string, bool) {
	relative, err := filepath.Rel(root, file)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", false
	}
	return relative, true
}

func relativeToModuleCache(file string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(file), "/pkg/mod/")
	if len(parts) < 2 {
		return "", false
	}
	return parts[len(parts)-1], true
}

func relativeToModule(file string) (string, bool) {
	root := moduleRoot(filepath.Dir(file))
	if root == "" {
		return "", false
	}
	modulePath := readModulePath(filepath.Join(root, "go.mod"))
	if modulePath == "" {
		return "", false
	}
	relative, ok := relativeToRoot(root, file)
	if !ok {
		return "", false
	}
	return modulePath + "/" + filepath.ToSlash(relative), true
}

func moduleRoot(dir string) string {
	for dir = filepath.Clean(dir); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			return ""
		}
	}
}

func readModulePath(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func formatFileLine(file string, line int) string {
	if line <= 0 {
		return file
	}
	return fmt.Sprintf("%s:%d", file, line)
}
