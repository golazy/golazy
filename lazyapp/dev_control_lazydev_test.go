//go:build lazydev

package lazyapp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyroutes"
)

type lazyDevReloadController struct {
	lazycontroller.Base
}

func newLazyDevReloadController(ctx context.Context) (*lazyDevReloadController, error) {
	base, err := lazycontroller.NewBase(ctx, "pages")
	if err != nil {
		return nil, err
	}
	return &lazyDevReloadController{Base: base}, nil
}

func (c *lazyDevReloadController) Index() error {
	return nil
}

func TestLazyDevControlPlaneReloadsViewsWithoutRebuildingApp(t *testing.T) {
	dir := t.TempDir()
	writeLazyDevControlFile(t, filepath.Join(dir, "layouts", "app.html.tpl"), `{{.content}}`)
	viewFile := filepath.Join(dir, "pages", "index.html.tpl")
	writeLazyDevControlFile(t, viewFile, `before`)

	previous := ViewsPath
	ViewsPath = dir
	t.Cleanup(func() {
		ViewsPath = previous
	})

	app := New(Config{
		Name: "test",
		Views: func() (fs.FS, error) {
			return nil, nil
		},
		Drawer: func(router *lazyroutes.Scope) {
			router.Get("/", newLazyDevReloadController, (*lazyDevReloadController).Index)
		},
	})
	if app.ControlPlane == nil {
		t.Fatal("lazydev app did not install a control plane")
	}

	assertLazyDevReloadBody(t, app, "before")
	writeLazyDevControlFile(t, viewFile, `after`)
	assertLazyDevReloadBody(t, app, "before")

	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/_golazy/views/reload", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("reload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Body.String(), "reload views ok\n"; got != want {
		t.Fatalf("reload body = %q, want %q", got, want)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("reload Cache-Control = %q, want no-store", got)
	}
	assertLazyDevReloadBody(t, app, "after")
}

func TestLazyDevControlPlaneOpensEditor(t *testing.T) {
	file := filepath.Join(t.TempDir(), "app", "controllers", "home.go")
	writeLazyDevControlFile(t, file, "package controllers\n")

	var gotName string
	var gotArgs []string
	previousStartEditorCommand := startEditorCommand
	startEditorCommand = func(name string, args ...string) error {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return nil
	}
	t.Cleanup(func() {
		startEditorCommand = previousStartEditorCommand
	})
	t.Setenv("EDITOR", "code --reuse-window")

	app := New(Config{Name: "test"})
	requestBody, err := json.Marshal(openEditorRequest{File: file, Line: 27})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodPost, lazyDevOpenEditorPath, bytes.NewReader(requestBody)))

	if response.Code != http.StatusNoContent {
		t.Fatalf("open editor status = %d, want %d: %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if gotName != "code" {
		t.Fatalf("editor command = %q, want code", gotName)
	}
	wantArgs := []string{"--reuse-window", "-g", file + ":27"}
	if !stringSlicesEqual(gotArgs, wantArgs) {
		t.Fatalf("editor args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestEditorCommandLineConventions(t *testing.T) {
	withoutDetectedTerminal(t)

	file := filepath.Join(t.TempDir(), "app name.go")
	tests := []struct {
		name     string
		editor   string
		wantName string
		wantArgs []string
	}{
		{
			name:     "code",
			editor:   "code --reuse-window",
			wantName: "code",
			wantArgs: []string{"--reuse-window", "-g", file + ":12"},
		},
		{
			name:     "nvim without terminal",
			editor:   "nvim -p",
			wantName: "nvim",
			wantArgs: []string{"-p", file, "+12"},
		},
		{
			name:     "emacs",
			editor:   "emacs",
			wantName: "emacs",
			wantArgs: []string{file, "+12"},
		},
		{
			name:     "unknown",
			editor:   "custom-editor --flag",
			wantName: "custom-editor",
			wantArgs: []string{"--flag", file},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotName, gotArgs, err := editorCommand(test.editor, file, 12)
			if err != nil {
				t.Fatal(err)
			}
			if gotName != test.wantName {
				t.Fatalf("name = %q, want %q", gotName, test.wantName)
			}
			if !stringSlicesEqual(gotArgs, test.wantArgs) {
				t.Fatalf("args = %#v, want %#v", gotArgs, test.wantArgs)
			}
		})
	}
}

func TestEditorCommandUsesLinuxTerminalFromEnvironment(t *testing.T) {
	isolateEditorCommandEnvironment(t)
	currentGOOS = "linux"
	t.Setenv("TERMINAL", "alacritty --class golazy")
	findExecutable = func(name string) (string, error) {
		if name == "alacritty" {
			return name, nil
		}
		return "", errors.New("not found")
	}

	file := filepath.Join(t.TempDir(), "app.go")
	gotName, gotArgs, err := editorCommand("nvim -p", file, 12)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "alacritty" {
		t.Fatalf("name = %q, want alacritty", gotName)
	}
	wantArgs := []string{"--class", "golazy", "-e", "nvim", "-p", file, "+12"}
	if !stringSlicesEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestEditorCommandUsesLinuxDefaultTerminal(t *testing.T) {
	isolateEditorCommandEnvironment(t)
	currentGOOS = "linux"
	findExecutable = func(name string) (string, error) {
		if name == "xdg-terminal-exec" {
			return name, nil
		}
		return "", errors.New("not found")
	}

	file := filepath.Join(t.TempDir(), "app.go")
	gotName, gotArgs, err := editorCommand("vim", file, 9)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "xdg-terminal-exec" {
		t.Fatalf("name = %q, want xdg-terminal-exec", gotName)
	}
	wantArgs := []string{"vim", file, "+9"}
	if !stringSlicesEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestEditorCommandUsesLinuxParentTerminal(t *testing.T) {
	isolateEditorCommandEnvironment(t)
	currentGOOS = "linux"
	processParent = func() int { return 20 }
	findExecutable = func(name string) (string, error) {
		if name == "kitty" {
			return name, nil
		}
		return "", errors.New("not found")
	}
	readProcLink = func(path string) (string, error) {
		if path == "/proc/10/exe" {
			return "/usr/bin/kitty", nil
		}
		return "", errors.New("not found")
	}
	readProcFile = func(path string) ([]byte, error) {
		switch path {
		case "/proc/20/comm":
			return []byte("bash\n"), nil
		case "/proc/20/stat":
			return []byte("20 (bash) S 10 0 0 0 0 0 0"), nil
		}
		return nil, errors.New("not found")
	}

	file := filepath.Join(t.TempDir(), "app.go")
	gotName, gotArgs, err := editorCommand("vim", file, 27)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "kitty" {
		t.Fatalf("name = %q, want kitty", gotName)
	}
	wantArgs := []string{"--", "vim", file, "+27"}
	if !stringSlicesEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestEditorCommandUsesMacTerminal(t *testing.T) {
	isolateEditorCommandEnvironment(t)
	currentGOOS = "darwin"

	file := filepath.Join(t.TempDir(), "app name.go")
	gotName, gotArgs, err := editorCommand("vim", file, 33)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "osascript" {
		t.Fatalf("name = %q, want osascript", gotName)
	}
	wantArgs := []string{
		"-e",
		"tell application \"Terminal\" to do script \"vim '" + file + "' +33\"",
		"-e",
		"tell application \"Terminal\" to activate",
	}
	if !stringSlicesEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func assertLazyDevReloadBody(t *testing.T, app *App, want string) {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("page status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Body.String(); got != want {
		t.Fatalf("page body = %q, want %q", got, want)
	}
}

func writeLazyDevControlFile(t *testing.T, filename string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func withoutDetectedTerminal(t *testing.T) {
	t.Helper()
	isolateEditorCommandEnvironment(t)
	currentGOOS = "linux"
	findExecutable = func(string) (string, error) { return "", errors.New("not found") }
	processParent = func() int { return 1 }
	readProcFile = func(string) ([]byte, error) { return nil, errors.New("not found") }
	readProcLink = func(string) (string, error) { return "", errors.New("not found") }
}

func isolateEditorCommandEnvironment(t *testing.T) {
	t.Helper()
	previousGOOS := currentGOOS
	previousFindExecutable := findExecutable
	previousProcessParent := processParent
	previousReadProcFile := readProcFile
	previousReadProcLink := readProcLink
	t.Setenv("TERMINAL", "")
	t.Cleanup(func() {
		currentGOOS = previousGOOS
		findExecutable = previousFindExecutable
		processParent = previousProcessParent
		readProcFile = previousReadProcFile
		readProcLink = previousReadProcLink
	})
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
