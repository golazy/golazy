// Package jsoncode provides structured JSON object edits as lazycode
// operations. It preserves the document's indentation and final newline but
// intentionally normalizes object key order through encoding/json.
package jsoncode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"golazy.dev/lazycode"
)

type Document struct {
	root         map[string]any
	indent       string
	finalNewline bool
}

type EditFunc func(*Document) (bool, error)

func Parse(data []byte) (*Document, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return nil, fmt.Errorf("jsoncode: parse: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return nil, errors.New("jsoncode: more than one JSON value")
	} else if !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("jsoncode: trailing data: %w", err)
	}
	indent := detectIndent(data)
	return &Document{
		root:         root,
		indent:       indent,
		finalNewline: len(data) > 0 && data[len(data)-1] == '\n',
	}, nil
}

func (d *Document) Bytes() ([]byte, error) {
	if d == nil {
		return nil, errors.New("jsoncode: nil document")
	}
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", d.indent)
	if err := encoder.Encode(d.root); err != nil {
		return nil, fmt.Errorf("jsoncode: encode: %w", err)
	}
	data := output.Bytes()
	if !d.finalNewline {
		data = bytes.TrimSuffix(data, []byte("\n"))
	}
	return append([]byte(nil), data...), nil
}

func (d *Document) Set(path []string, value any) (bool, error) {
	if d == nil {
		return false, errors.New("jsoncode: nil document")
	}
	if len(path) == 0 {
		return false, errors.New("jsoncode: key path is required")
	}
	object := d.root
	for index, key := range path[:len(path)-1] {
		if key == "" {
			return false, fmt.Errorf("jsoncode: key path element %d is empty", index)
		}
		next, ok := object[key]
		if !ok {
			created := make(map[string]any)
			object[key] = created
			object = created
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return false, fmt.Errorf("jsoncode: %s is not an object", strings.Join(path[:index+1], "."))
		}
		object = child
	}
	key := path[len(path)-1]
	if key == "" {
		return false, errors.New("jsoncode: final key is empty")
	}
	normalized, err := normalize(value)
	if err != nil {
		return false, err
	}
	if current, ok := object[key]; ok && equalJSON(current, normalized) {
		return false, nil
	}
	object[key] = normalized
	return true, nil
}

func (d *Document) Remove(path []string) (bool, error) {
	if d == nil {
		return false, errors.New("jsoncode: nil document")
	}
	if len(path) == 0 {
		return false, errors.New("jsoncode: key path is required")
	}
	object := d.root
	for index, key := range path[:len(path)-1] {
		next, ok := object[key]
		if !ok {
			return false, nil
		}
		child, ok := next.(map[string]any)
		if !ok {
			return false, fmt.Errorf("jsoncode: %s is not an object", strings.Join(path[:index+1], "."))
		}
		object = child
	}
	key := path[len(path)-1]
	if _, ok := object[key]; !ok {
		return false, nil
	}
	delete(object, key)
	return true, nil
}

func (d *Document) EnsureDependency(group, name, version string) (bool, error) {
	if !validDependencyGroup(group) {
		return false, fmt.Errorf("jsoncode: unsupported dependency group %q", group)
	}
	if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return false, errors.New("jsoncode: dependency name and version are required")
	}
	return d.Set([]string{group, name}, version)
}

func (d *Document) RemoveDependency(group, name string) (bool, error) {
	if !validDependencyGroup(group) {
		return false, fmt.Errorf("jsoncode: unsupported dependency group %q", group)
	}
	return d.Remove([]string{group, name})
}

func Edit(name string, edit EditFunc) lazycode.Operation {
	return lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		if edit == nil {
			return errors.New("jsoncode: edit function is required")
		}
		source, err := workspace.Read(name)
		if err != nil {
			return err
		}
		document, err := Parse(source)
		if err != nil {
			return fmt.Errorf("jsoncode: parse %s: %w", name, err)
		}
		changed, err := edit(document)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		data, err := document.Bytes()
		if err != nil {
			return err
		}
		return workspace.Replace(name, data)
	})
}

func Set(name string, path []string, value any) lazycode.Operation {
	return Edit(name, func(document *Document) (bool, error) {
		return document.Set(path, value)
	})
}

func Dependency(name, group, dependency, version string) lazycode.Operation {
	return Edit(name, func(document *Document) (bool, error) {
		return document.EnsureDependency(group, dependency, version)
	})
}

func normalize(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("jsoncode: unsupported value: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("jsoncode: normalize value: %w", err)
	}
	return normalized, nil
}

func equalJSON(left, right any) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftData, rightData)
}

func detectIndent(data []byte) string {
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines[1:] {
		trimmed := bytes.TrimLeft(line, " \t")
		if len(trimmed) == 0 || len(trimmed) == len(line) {
			continue
		}
		return string(line[:len(line)-len(trimmed)])
	}
	return "  "
}

func validDependencyGroup(group string) bool {
	switch group {
	case "dependencies", "devDependencies", "peerDependencies", "optionalDependencies":
		return true
	default:
		return false
	}
}
