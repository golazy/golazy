// Package lazycode plans source-file changes without writing them.
//
// Format-specific packages build Operations that edit an in-memory Workspace.
// Plan returns baseline-bound edits for a caller, such as the lazy CLI, to
// review and apply transactionally.
package lazycode

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const AbsentHash = "absent"

type EditKind string

const (
	EditCreate EditKind = "create"
	EditUpdate EditKind = "update"
	EditDelete EditKind = "delete"
)

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Diagnostic struct {
	Path     string
	Line     int
	Column   int
	Message  string
	Severity Severity
}

type FileEdit struct {
	Path         string
	Kind         EditKind
	BaselineHash string
	Before       []byte
	After        []byte
}

// VerifyBaseline reports whether data still matches the input used to plan e.
func (e FileEdit) VerifyBaseline(data []byte, exists bool) bool {
	if e.BaselineHash == AbsentHash {
		return !exists
	}
	return exists && e.BaselineHash == Hash(data)
}

type Result struct {
	Files       []FileEdit
	Diagnostics []Diagnostic
}

func (r Result) Changed() bool { return len(r.Files) != 0 }

type Operation interface {
	Apply(*Workspace) error
}

type OperationFunc func(*Workspace) error

func (f OperationFunc) Apply(workspace *Workspace) error {
	if f == nil {
		return errors.New("lazycode: nil operation")
	}
	return f(workspace)
}

type fileState struct {
	data   []byte
	exists bool
}

// Workspace holds an immutable baseline and an in-memory working copy. Its
// mutation methods never touch the filesystem and are intended for Operations.
type Workspace struct {
	root        string
	baseline    map[string]fileState
	files       map[string]fileState
	diagnostics []Diagnostic
}

func New(root string) *Workspace {
	return &Workspace{
		root:     root,
		baseline: make(map[string]fileState),
		files:    make(map[string]fileState),
	}
}

func FromFiles(root string, files map[string][]byte) (*Workspace, error) {
	workspace := New(root)
	paths := make([]string, 0, len(files))
	for name := range files {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	for _, name := range paths {
		if err := workspace.Add(name, files[name]); err != nil {
			return nil, err
		}
	}
	return workspace, nil
}

// Load reads relative paths into a Workspace. It never writes to root.
func Load(root string, paths ...string) (*Workspace, error) {
	workspace := New(root)
	for _, name := range paths {
		clean, err := cleanPath(name)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(clean)))
		if err != nil {
			return nil, fmt.Errorf("lazycode: read %s: %w", clean, err)
		}
		if err := workspace.Add(clean, data); err != nil {
			return nil, err
		}
	}
	return workspace, nil
}

func (w *Workspace) Root() string {
	if w == nil {
		return ""
	}
	return w.root
}

// Add adds an existing file to the immutable baseline.
func (w *Workspace) Add(name string, data []byte) error {
	if w == nil {
		return errors.New("lazycode: nil workspace")
	}
	clean, err := cleanPath(name)
	if err != nil {
		return err
	}
	if state, ok := w.baseline[clean]; ok && state.exists {
		return fmt.Errorf("lazycode: baseline file %q already exists", clean)
	}
	state := fileState{data: cloneBytes(data), exists: true}
	w.baseline[clean] = state
	w.files[clean] = fileState{data: cloneBytes(data), exists: true}
	return nil
}

func (w *Workspace) Read(name string) ([]byte, error) {
	if w == nil {
		return nil, errors.New("lazycode: nil workspace")
	}
	clean, err := cleanPath(name)
	if err != nil {
		return nil, err
	}
	state, ok := w.files[clean]
	if !ok || !state.exists {
		return nil, fmt.Errorf("lazycode: read %s: %w", clean, fs.ErrNotExist)
	}
	return cloneBytes(state.data), nil
}

func (w *Workspace) Exists(name string) bool {
	clean, err := cleanPath(name)
	if w == nil || err != nil {
		return false
	}
	state, ok := w.files[clean]
	return ok && state.exists
}

// Replace creates or replaces a file in memory.
func (w *Workspace) Replace(name string, data []byte) error {
	if w == nil {
		return errors.New("lazycode: nil workspace")
	}
	clean, err := cleanPath(name)
	if err != nil {
		return err
	}
	w.files[clean] = fileState{data: cloneBytes(data), exists: true}
	return nil
}

// Remove marks a file absent in memory.
func (w *Workspace) Remove(name string) error {
	if w == nil {
		return errors.New("lazycode: nil workspace")
	}
	clean, err := cleanPath(name)
	if err != nil {
		return err
	}
	w.files[clean] = fileState{exists: false}
	return nil
}

func (w *Workspace) Diagnose(diagnostic Diagnostic) error {
	if w == nil {
		return errors.New("lazycode: nil workspace")
	}
	if diagnostic.Message == "" {
		return errors.New("lazycode: diagnostic message is required")
	}
	if diagnostic.Path != "" {
		clean, err := cleanPath(diagnostic.Path)
		if err != nil {
			return err
		}
		diagnostic.Path = clean
	}
	if diagnostic.Severity == "" {
		diagnostic.Severity = SeverityInfo
	}
	w.diagnostics = append(w.diagnostics, diagnostic)
	return nil
}

func (w *Workspace) Paths() []string {
	if w == nil {
		return nil
	}
	paths := make([]string, 0, len(w.files))
	for name, state := range w.files {
		if state.exists {
			paths = append(paths, name)
		}
	}
	sort.Strings(paths)
	return paths
}

// Plan applies operations to a fresh clone, leaving w reusable and unchanged.
func (w *Workspace) Plan(operations ...Operation) (Result, error) {
	if w == nil {
		return Result{}, errors.New("lazycode: nil workspace")
	}
	working := w.cloneBaseline()
	for index, operation := range operations {
		if operation == nil {
			return Result{}, fmt.Errorf("lazycode: operation %d is nil", index+1)
		}
		if err := operation.Apply(working); err != nil {
			return Result{}, fmt.Errorf("lazycode: operation %d: %w", index+1, err)
		}
	}
	return working.result(), nil
}

func Hash(data []byte) string {
	digest := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", digest)
}

func (w *Workspace) cloneBaseline() *Workspace {
	clone := New(w.root)
	for name, state := range w.baseline {
		clone.baseline[name] = cloneState(state)
		clone.files[name] = cloneState(state)
	}
	return clone
}

func (w *Workspace) result() Result {
	names := make(map[string]struct{}, len(w.baseline)+len(w.files))
	for name := range w.baseline {
		names[name] = struct{}{}
	}
	for name := range w.files {
		names[name] = struct{}{}
	}
	paths := make([]string, 0, len(names))
	for name := range names {
		paths = append(paths, name)
	}
	sort.Strings(paths)

	result := Result{Diagnostics: append([]Diagnostic(nil), w.diagnostics...)}
	for _, name := range paths {
		before := w.baseline[name]
		after := w.files[name]
		if before.exists == after.exists && (!before.exists || string(before.data) == string(after.data)) {
			continue
		}
		edit := FileEdit{
			Path:   name,
			Before: cloneBytes(before.data),
			After:  cloneBytes(after.data),
		}
		switch {
		case !before.exists && after.exists:
			edit.Kind = EditCreate
			edit.BaselineHash = AbsentHash
		case before.exists && !after.exists:
			edit.Kind = EditDelete
			edit.BaselineHash = Hash(before.data)
		default:
			edit.Kind = EditUpdate
			edit.BaselineHash = Hash(before.data)
		}
		result.Files = append(result.Files, edit)
	}
	return result
}

func cleanPath(name string) (string, error) {
	if name == "" {
		return "", errors.New("lazycode: path is required")
	}
	name = filepath.ToSlash(name)
	if strings.HasPrefix(name, "/") || filepath.IsAbs(name) {
		return "", fmt.Errorf("lazycode: path %q must be relative", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || !fs.ValidPath(clean) {
		return "", fmt.Errorf("lazycode: invalid path %q", name)
	}
	return clean, nil
}

func cloneState(state fileState) fileState {
	return fileState{data: cloneBytes(state.data), exists: state.exists}
}

func cloneBytes(data []byte) []byte {
	return append([]byte(nil), data...)
}
