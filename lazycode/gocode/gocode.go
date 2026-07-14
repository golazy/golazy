package gocode

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"golazy.dev/lazycode"
)

// RewriteFunc mutates a parsed Go file and reports whether it changed.
type RewriteFunc func(*token.FileSet, *ast.File) (bool, error)

// Parse parses source as a Go file, preserving comments and skipping deprecated
// object resolution.
func Parse(name string, source []byte) (*token.FileSet, *ast.File, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, name, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, fmt.Errorf("gocode: parse %s: %w", name, err)
	}
	return fileSet, file, nil
}

// RewriteSource applies rewrite, formats the syntax tree, and reports whether
// the formatted result differs from source.
func RewriteSource(name string, source []byte, rewrite RewriteFunc) ([]byte, bool, error) {
	if rewrite == nil {
		return nil, false, errors.New("gocode: rewrite function is required")
	}
	fileSet, file, err := Parse(name, source)
	if err != nil {
		return nil, false, err
	}
	changed, err := rewrite(fileSet, file)
	if err != nil {
		return nil, false, err
	}
	if !changed {
		return append([]byte(nil), source...), false, nil
	}
	var formatted bytes.Buffer
	if err := format.Node(&formatted, fileSet, file); err != nil {
		return nil, false, fmt.Errorf("gocode: format %s: %w", name, err)
	}
	if bytes.Equal(source, formatted.Bytes()) {
		return append([]byte(nil), source...), false, nil
	}
	return formatted.Bytes(), true, nil
}

// Rewrite returns an operation that rewrites a Go file in memory.
func Rewrite(name string, rewrite RewriteFunc) lazycode.Operation {
	return lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		source, err := workspace.Read(name)
		if err != nil {
			return err
		}
		result, changed, err := RewriteSource(name, source, rewrite)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		return workspace.Replace(name, result)
	})
}

// EnsureFile formats source and creates or replaces name in memory. It is
// useful for generated Go sidecars and rejects syntactically invalid source.
func EnsureFile(name string, source []byte) lazycode.Operation {
	return lazycode.OperationFunc(func(workspace *lazycode.Workspace) error {
		formatted, err := format.Source(source)
		if err != nil {
			return fmt.Errorf("gocode: format %s: %w", name, err)
		}
		if existing, err := workspace.Read(name); err == nil && bytes.Equal(existing, formatted) {
			return nil
		}
		return workspace.Replace(name, formatted)
	})
}

// EnsureImport adds an ordinary import if it is absent and reports whether the
// syntax tree changed. Use EnsureNamedImport when validation errors must be
// distinguished from an unchanged file.
func EnsureImport(file *ast.File, importPath string) bool {
	changed, _ := ensureNamedImport(file, "", importPath)
	return changed
}

// EnsureBlankImport adds importPath with the blank identifier.
func EnsureBlankImport(file *ast.File, importPath string) (bool, error) {
	return ensureNamedImport(file, "_", importPath)
}

// EnsureNamedImport adds importPath with name. An empty name creates an
// ordinary import; "_" and "." create blank and dot imports respectively.
func EnsureNamedImport(file *ast.File, name, importPath string) (bool, error) {
	if name == "_" {
		return EnsureBlankImport(file, importPath)
	}
	if name == "." {
		return ensureNamedImport(file, name, importPath)
	}
	if name != "" && !token.IsIdentifier(name) {
		return false, fmt.Errorf("gocode: invalid import name %q", name)
	}
	return ensureNamedImport(file, name, importPath)
}

func ensureNamedImport(file *ast.File, name, importPath string) (bool, error) {
	if file == nil {
		return false, errors.New("gocode: nil Go file")
	}
	if strings.TrimSpace(importPath) == "" {
		return false, errors.New("gocode: import path is required")
	}
	for _, spec := range importSpecs(file) {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		if path != importPath {
			if name != "" && name != "_" && spec.Name != nil && spec.Name.Name == name {
				return false, fmt.Errorf("gocode: import name %q already used by %q", name, path)
			}
			continue
		}
		existing := ""
		if spec.Name != nil {
			existing = spec.Name.Name
		}
		if existing != name {
			return false, fmt.Errorf("gocode: import %q already uses name %q, not %q", importPath, existing, name)
		}
		return false, nil
	}

	spec := &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(importPath)}}
	if name != "" {
		spec.Name = ast.NewIdent(name)
	}
	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			continue
		}
		imports.Specs = append(imports.Specs, spec)
		return true, nil
	}
	file.Decls = append([]ast.Decl{&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{spec}}}, file.Decls...)
	return true, nil
}

// RemoveImport removes every declaration of importPath and reports whether the
// syntax tree changed.
func RemoveImport(file *ast.File, importPath string) bool {
	if file == nil || importPath == "" {
		return false
	}
	changed := false
	declarations := make([]ast.Decl, 0, len(file.Decls))
	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			declarations = append(declarations, declaration)
			continue
		}
		specs := make([]ast.Spec, 0, len(imports.Specs))
		for _, raw := range imports.Specs {
			spec, ok := raw.(*ast.ImportSpec)
			if !ok {
				specs = append(specs, raw)
				continue
			}
			path, err := strconv.Unquote(spec.Path.Value)
			if err == nil && path == importPath {
				changed = true
				continue
			}
			specs = append(specs, spec)
		}
		if len(specs) == 0 {
			changed = true
			continue
		}
		imports.Specs = specs
		declarations = append(declarations, imports)
	}
	if changed {
		file.Decls = declarations
	}
	return changed
}

// HasImport reports whether file imports importPath with any import name.
func HasImport(file *ast.File, importPath string) bool {
	for _, spec := range importSpecs(file) {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == importPath {
			return true
		}
	}
	return false
}

// ImportName reports the explicit name for importPath. An empty name means a
// normal import whose package name is inferred by Go. The returned error
// rejects malformed or duplicate imports instead of choosing one silently.
func ImportName(file *ast.File, importPath string) (string, bool, error) {
	if file == nil {
		return "", false, errors.New("gocode: nil Go file")
	}
	found := false
	name := ""
	for _, spec := range importSpecs(file) {
		candidate, err := strconv.Unquote(spec.Path.Value)
		if err != nil || candidate != importPath {
			continue
		}
		if found {
			return "", false, fmt.Errorf("gocode: import %q is declared more than once", importPath)
		}
		found = true
		if spec.Name != nil {
			name = spec.Name.Name
		}
	}
	return name, found, nil
}

// UsesSelector reports whether file contains a selector rooted at packageName,
// such as seo.Configure.
func UsesSelector(file *ast.File, packageName string) bool {
	if file == nil || packageName == "" {
		return false
	}
	used := false
	ast.Inspect(file, func(node ast.Node) bool {
		if used {
			return false
		}
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, ok := selector.X.(*ast.Ident)
		if ok && identifier.Name == packageName {
			used = true
			return false
		}
		return true
	})
	return used
}

func importSpecs(file *ast.File) []*ast.ImportSpec {
	if file == nil {
		return nil
	}
	var specs []*ast.ImportSpec
	for _, declaration := range file.Decls {
		imports, ok := declaration.(*ast.GenDecl)
		if !ok || imports.Tok != token.IMPORT {
			continue
		}
		for _, raw := range imports.Specs {
			if spec, ok := raw.(*ast.ImportSpec); ok {
				specs = append(specs, spec)
			}
		}
	}
	return specs
}
