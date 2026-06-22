package lazydoc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func LoadDir(dir, version string) (*Index, error) {
	modulePath, err := readModulePath(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, err
	}
	packages, err := LoadPackagesFromDir(dir, modulePath)
	if err != nil {
		return nil, err
	}
	return &Index{Versions: []Version{{
		Version:  version,
		Module:   modulePath,
		Packages: packages,
	}}}, nil
}

func LoadPackagesFromDir(dir, modulePath string) ([]Package, error) {
	var packages []Package
	if err := filepath.WalkDir(dir, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if shouldSkipDir(entry.Name()) && current != dir {
			return filepath.SkipDir
		}
		pkg, ok, err := loadPackageDir(current, dir, modulePath)
		if err != nil {
			return err
		}
		if ok {
			packages = append(packages, pkg)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Slice(packages, func(i, j int) bool {
		return packages[i].ImportPath < packages[j].ImportPath
	})
	return packages, nil
}

func loadPackageDir(dir, moduleRoot, modulePath string) (Package, bool, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return Package{}, false, err
	}
	var goFiles []string
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		goFiles = append(goFiles, filepath.Join(dir, file.Name()))
	}
	if len(goFiles) == 0 {
		return Package{}, false, nil
	}

	fset := token.NewFileSet()
	parsed := make([]*ast.File, 0, len(goFiles))
	for _, file := range goFiles {
		astFile, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return Package{}, false, fmt.Errorf("parse %s: %w", file, err)
		}
		if strings.HasSuffix(astFile.Name.Name, "_test") {
			continue
		}
		parsed = append(parsed, astFile)
	}
	if len(parsed) == 0 {
		return Package{}, false, nil
	}

	rel, err := filepath.Rel(moduleRoot, dir)
	if err != nil {
		return Package{}, false, err
	}
	importPath := modulePath
	if rel != "." {
		importPath += "/" + filepath.ToSlash(rel)
	}
	docPackage, err := doc.NewFromFiles(fset, parsed, importPath, doc.Mode(0))
	if err != nil {
		return Package{}, false, fmt.Errorf("build docs for %s: %w", dir, err)
	}
	return convertPackage(fset, importPath, docPackage), true, nil
}

func convertPackage(fset *token.FileSet, importPath string, pkg *doc.Package) Package {
	out := Package{
		ImportPath: importPath,
		Name:       pkg.Name,
		Synopsis:   doc.Synopsis(pkg.Doc),
		Doc:        strings.TrimSpace(pkg.Doc),
		Constants:  convertValues(fset, pkg.Consts),
		Variables:  convertValues(fset, pkg.Vars),
		Functions:  convertFuncs(fset, pkg.Funcs, pkg.Examples),
		Types:      convertTypes(fset, pkg.Types),
		Examples:   convertExamples(fset, pkg.Examples, ""),
	}
	return out
}

func convertValues(fset *token.FileSet, values []*doc.Value) []Value {
	out := make([]Value, 0, len(values))
	for _, value := range values {
		out = append(out, Value{
			Names: append([]string(nil), value.Names...),
			Doc:   strings.TrimSpace(value.Doc),
			Decl:  nodeString(fset, value.Decl),
		})
	}
	return out
}

func convertFuncs(fset *token.FileSet, funcs []*doc.Func, examples []*doc.Example) []Func {
	out := make([]Func, 0, len(funcs))
	for _, fn := range funcs {
		out = append(out, Func{
			Name:     fn.Name,
			Doc:      strings.TrimSpace(fn.Doc),
			Decl:     nodeString(fset, fn.Decl),
			Examples: convertExamples(fset, examples, fn.Name),
		})
	}
	return out
}

func convertTypes(fset *token.FileSet, types []*doc.Type) []Type {
	out := make([]Type, 0, len(types))
	for _, typ := range types {
		out = append(out, Type{
			Name:      typ.Name,
			Doc:       strings.TrimSpace(typ.Doc),
			Decl:      nodeString(fset, typ.Decl),
			Constants: convertValues(fset, typ.Consts),
			Variables: convertValues(fset, typ.Vars),
			Funcs:     convertFuncs(fset, typ.Funcs, typ.Examples),
			Methods:   convertFuncs(fset, typ.Methods, typ.Examples),
			Examples:  convertExamples(fset, typ.Examples, typ.Name),
		})
	}
	return out
}

func convertExamples(fset *token.FileSet, examples []*doc.Example, name string) []Example {
	var out []Example
	for _, example := range examples {
		if example.Name != name {
			continue
		}
		out = append(out, Example{
			Name:   example.Name,
			Suffix: example.Suffix,
			Doc:    strings.TrimSpace(example.Doc),
			Code:   exampleCode(fset, example),
			Output: strings.TrimSpace(example.Output),
		})
	}
	return out
}

func exampleCode(fset *token.FileSet, example *doc.Example) string {
	if example == nil || example.Code == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, example.Code); err == nil {
		return strings.TrimSpace(buf.String())
	}
	return nodeString(fset, example.Code)
}

func nodeString(fset *token.FileSet, node any) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, node); err != nil {
		return ""
	}
	return strings.TrimSpace(buf.String())
}

func readModulePath(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module path not found in go.mod")
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".tmp", "node_modules", "vendor", "testdata":
		return true
	}
	return strings.HasPrefix(name, ".")
}
