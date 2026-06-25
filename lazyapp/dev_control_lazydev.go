//go:build lazydev

package lazyapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazycontrolplane"
)

const lazyDevReloadViewsPath = "/_golazy/views/reload"
const lazyDevOpenEditorPath = "/_golazy/open-editor"

var startEditorCommand = startEditorCommandDefault

var (
	currentGOOS    = runtime.GOOS
	findExecutable = exec.LookPath
	processParent  = os.Getppid
	readProcFile   = os.ReadFile
	readProcLink   = os.Readlink
)

func lazyDevContext(ctx context.Context) context.Context {
	return lazycontroller.WithOpenEditorPath(ctx, lazyDevOpenEditorPath)
}

func lazyDevControlPlane(controlPlane *lazycontrolplane.ControlPlane, renderer *lazycontroller.Renderer) *lazycontrolplane.ControlPlane {
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	var reloadMu sync.Mutex
	controlPlane.Handle("POST "+lazyDevReloadViewsPath, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reloadMu.Lock()
		defer reloadMu.Unlock()

		if renderer == nil {
			writeLazyDevControlResponse(w, http.StatusInternalServerError, "reload views: renderer is missing\n")
			return
		}
		if err := renderer.Cache(); err != nil {
			writeLazyDevControlResponse(w, http.StatusInternalServerError, fmt.Sprintf("reload views: %v\n", err))
			return
		}
		writeLazyDevControlResponse(w, http.StatusOK, "reload views ok\n")
	}))
	controlPlane.Handle("POST "+lazyDevOpenEditorPath, http.HandlerFunc(openEditor))
	return controlPlane
}

type openEditorRequest struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func openEditor(w http.ResponseWriter, r *http.Request) {
	var request openEditorRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
	if err := decoder.Decode(&request); err != nil {
		writeLazyDevControlResponse(w, http.StatusBadRequest, fmt.Sprintf("open editor: invalid request: %v\n", err))
		return
	}

	file := filepath.Clean(strings.TrimSpace(request.File))
	if file == "" || !filepath.IsAbs(file) {
		writeLazyDevControlResponse(w, http.StatusBadRequest, "open editor: file must be absolute\n")
		return
	}
	if request.Line <= 0 {
		writeLazyDevControlResponse(w, http.StatusBadRequest, "open editor: line must be positive\n")
		return
	}
	info, err := os.Stat(file)
	if err != nil {
		writeLazyDevControlResponse(w, http.StatusBadRequest, fmt.Sprintf("open editor: %v\n", err))
		return
	}
	if info.IsDir() {
		writeLazyDevControlResponse(w, http.StatusBadRequest, "open editor: file is a directory\n")
		return
	}

	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		writeLazyDevControlResponse(w, http.StatusBadRequest, "open editor: EDITOR is not set\n")
		return
	}
	name, args, err := editorCommand(editor, file, request.Line)
	if err != nil {
		writeLazyDevControlResponse(w, http.StatusBadRequest, fmt.Sprintf("open editor: %v\n", err))
		return
	}
	if err := startEditorCommand(name, args...); err != nil {
		writeLazyDevControlResponse(w, http.StatusInternalServerError, fmt.Sprintf("open editor: %v\n", err))
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func editorCommand(editor string, file string, line int) (string, []string, error) {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("EDITOR is empty")
	}

	name := parts[0]
	args := append([]string(nil), parts[1:]...)
	editorName := strings.ToLower(strings.TrimSuffix(filepath.Base(name), ".exe"))
	lineArg := fmt.Sprintf("+%d", line)

	switch {
	case strings.Contains(editorName, "code"):
		if !containsCodeGotoFlag(args) {
			args = append(args, "-g")
		}
		args = append(args, fmt.Sprintf("%s:%d", file, line))
	case isTerminalEditor(editorName):
		args = append(args, file, lineArg)
		if terminalName, terminalArgs, ok := terminalCommand([]string{name}, args); ok {
			return terminalName, terminalArgs, nil
		}
	case editorName == "emacs":
		args = append(args, file, lineArg)
	default:
		args = append(args, file)
	}
	return name, args, nil
}

func containsCodeGotoFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-g" || arg == "--goto" {
			return true
		}
	}
	return false
}

func isTerminalEditor(editorName string) bool {
	switch editorName {
	case "vim", "nvim", "neovim", "nano":
		return true
	default:
		return false
	}
}

func terminalCommand(command []string, args []string) (string, []string, bool) {
	fullCommand := append(append([]string(nil), command...), args...)
	switch currentGOOS {
	case "darwin":
		script := "tell application \"Terminal\" to do script " + appleScriptString(shellJoin(fullCommand))
		return "osascript", []string{"-e", script, "-e", "tell application \"Terminal\" to activate"}, true
	case "linux":
		terminal, ok := linuxTerminal()
		if !ok {
			return "", nil, false
		}
		return terminalLaunchCommand(terminal, fullCommand)
	default:
		return "", nil, false
	}
}

type terminalSpec struct {
	Name string
	Args []string
}

func linuxTerminal() (terminalSpec, bool) {
	if terminal, ok := terminalFromEnvironment(); ok {
		return terminal, true
	}
	for _, name := range []string{"xdg-terminal-exec", "x-terminal-emulator"} {
		if terminal, ok := terminalFromName(name); ok {
			return terminal, true
		}
	}
	if terminal, ok := linuxParentTerminal(); ok {
		return terminal, true
	}
	for _, name := range linuxTerminalNames() {
		if terminal, ok := terminalFromName(name); ok {
			return terminal, true
		}
	}
	return terminalSpec{}, false
}

func terminalFromEnvironment() (terminalSpec, bool) {
	parts := strings.Fields(strings.TrimSpace(os.Getenv("TERMINAL")))
	if len(parts) == 0 {
		return terminalSpec{}, false
	}
	if _, err := findExecutable(parts[0]); err != nil {
		return terminalSpec{}, false
	}
	return terminalSpec{Name: parts[0], Args: append([]string(nil), parts[1:]...)}, true
}

func terminalFromName(name string) (terminalSpec, bool) {
	if _, err := findExecutable(name); err != nil {
		return terminalSpec{}, false
	}
	return terminalSpec{Name: name}, true
}

func terminalLaunchCommand(terminal terminalSpec, command []string) (string, []string, bool) {
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(terminal.Name), ".exe"))
	args := append([]string(nil), terminal.Args...)
	switch name {
	case "xdg-terminal-exec":
		args = append(args, command...)
	case "wezterm":
		args = append(args, "start", "--")
		args = append(args, command...)
	case "gnome-terminal", "gnome-console", "kgx", "ptyxis", "mate-terminal", "kitty", "foot":
		args = append(args, "--")
		args = append(args, command...)
	case "xfce4-terminal":
		args = append(args, "-x")
		args = append(args, command...)
	default:
		args = append(args, "-e")
		args = append(args, command...)
	}
	return terminal.Name, args, true
}

func linuxTerminalNames() []string {
	return []string{
		"gnome-terminal",
		"kgx",
		"ptyxis",
		"konsole",
		"xfce4-terminal",
		"mate-terminal",
		"kitty",
		"alacritty",
		"wezterm",
		"ghostty",
		"foot",
		"tilix",
		"terminator",
		"lxterminal",
		"xterm",
		"urxvt",
		"rxvt",
		"st",
		"stterm",
	}
}

func linuxParentTerminal() (terminalSpec, bool) {
	pid := processParent()
	for range 32 {
		if pid <= 1 {
			return terminalSpec{}, false
		}
		if terminal, ok := linuxProcessTerminal(pid); ok {
			return terminal, true
		}
		parent, err := linuxProcessParent(pid)
		if err != nil || parent == pid {
			return terminalSpec{}, false
		}
		pid = parent
	}
	return terminalSpec{}, false
}

func linuxProcessTerminal(pid int) (terminalSpec, bool) {
	executable, _ := readProcLink(fmt.Sprintf("/proc/%d/exe", pid))
	if terminal, ok := linuxTerminalFromProcessName(executable, executable); ok {
		return terminal, true
	}
	data, err := readProcFile(fmt.Sprintf("/proc/%d/comm", pid))
	if err != nil {
		return terminalSpec{}, false
	}
	return linuxTerminalFromProcessName(string(data), executable)
}

func linuxTerminalFromProcessName(processName string, executable string) (terminalSpec, bool) {
	name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(filepath.Base(processName), ".exe")))
	if name == "" {
		return terminalSpec{}, false
	}
	terminalName, ok := linuxParentTerminalCommands()[name]
	if !ok {
		return terminalSpec{}, false
	}
	if terminal, ok := terminalFromName(terminalName); ok {
		return terminal, true
	}
	if executable != "" && name == terminalName {
		return terminalSpec{Name: executable}, true
	}
	return terminalSpec{}, false
}

func linuxParentTerminalCommands() map[string]string {
	return map[string]string{
		"alacritty":             "alacritty",
		"contour":               "contour",
		"deepin-terminal":       "deepin-terminal",
		"foot":                  "foot",
		"ghostty":               "ghostty",
		"gnome-console":         "gnome-console",
		"gnome-terminal":        "gnome-terminal",
		"gnome-terminal-server": "gnome-terminal",
		"kgx":                   "kgx",
		"kitty":                 "kitty",
		"konsole":               "konsole",
		"lxterminal":            "lxterminal",
		"mate-terminal":         "mate-terminal",
		"ptyxis":                "ptyxis",
		"rio":                   "rio",
		"rxvt":                  "rxvt",
		"st":                    "st",
		"stterm":                "stterm",
		"terminator":            "terminator",
		"terminology":           "terminology",
		"tilix":                 "tilix",
		"urxvt":                 "urxvt",
		"wezterm":               "wezterm",
		"wezterm-gui":           "wezterm",
		"xfce4-terminal":        "xfce4-terminal",
		"xterm":                 "xterm",
	}
}

func linuxProcessParent(pid int) (int, error) {
	data, err := readProcFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, err
	}
	text := string(data)
	nameEnd := strings.LastIndex(text, ")")
	if nameEnd == -1 || nameEnd+1 >= len(text) {
		return 0, fmt.Errorf("invalid proc stat")
	}
	fields := strings.Fields(text[nameEnd+1:])
	if len(fields) < 2 {
		return 0, fmt.Errorf("invalid proc stat")
	}
	var parent int
	if _, err := fmt.Sscanf(fields[1], "%d", &parent); err != nil {
		return 0, err
	}
	return parent, nil
}

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for index, arg := range args {
		quoted[index] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	if isShellSafe(arg) {
		return arg
	}
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}

func isShellSafe(arg string) bool {
	for _, char := range arg {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '/', '.', '_', '-', ':', '=', '+', ',', '%', '@':
			continue
		default:
			return false
		}
	}
	return true
}

func startEditorCommandDefault(name string, args ...string) error {
	command := exec.Command(name, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		return err
	}
	go func() {
		_ = command.Wait()
	}()
	return nil
}

func writeLazyDevControlResponse(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}
