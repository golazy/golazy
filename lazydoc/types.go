package lazydoc

import "strings"

type Index struct {
	Versions []Version `json:"versions"`
}

type Version struct {
	Version  string    `json:"version"`
	Module   string    `json:"module"`
	Packages []Package `json:"packages"`
}

type Package struct {
	ImportPath string    `json:"import_path"`
	Name       string    `json:"name"`
	Synopsis   string    `json:"synopsis"`
	Doc        string    `json:"doc"`
	Constants  []Value   `json:"constants,omitempty"`
	Variables  []Value   `json:"variables,omitempty"`
	Functions  []Func    `json:"functions,omitempty"`
	Types      []Type    `json:"types,omitempty"`
	Examples   []Example `json:"examples,omitempty"`
}

type Value struct {
	Names []string `json:"names"`
	Doc   string   `json:"doc"`
	Decl  string   `json:"decl"`
}

type Func struct {
	Name     string    `json:"name"`
	Doc      string    `json:"doc"`
	Decl     string    `json:"decl"`
	Examples []Example `json:"examples,omitempty"`
}

type Type struct {
	Name      string    `json:"name"`
	Doc       string    `json:"doc"`
	Decl      string    `json:"decl"`
	Constants []Value   `json:"constants,omitempty"`
	Variables []Value   `json:"variables,omitempty"`
	Funcs     []Func    `json:"functions,omitempty"`
	Methods   []Func    `json:"methods,omitempty"`
	Examples  []Example `json:"examples,omitempty"`
}

type Example struct {
	Name   string `json:"name"`
	Suffix string `json:"suffix,omitempty"`
	Doc    string `json:"doc,omitempty"`
	Code   string `json:"code"`
	Output string `json:"output,omitempty"`
}

type SearchResult struct {
	Version     string
	PackagePath string
	PackageName string
	Kind        string
	Name        string
	URL         string
	Summary     string
}

func (i *Index) Version(value string) (*Version, bool) {
	if i == nil {
		return nil, false
	}
	for index := range i.Versions {
		if i.Versions[index].Version == value {
			return &i.Versions[index], true
		}
	}
	return nil, false
}

func (i *Index) Latest() (*Version, bool) {
	if i == nil || len(i.Versions) == 0 {
		return nil, false
	}
	return &i.Versions[0], true
}

func (v *Version) Package(pathValue string) (*Package, bool) {
	if v == nil {
		return nil, false
	}
	for index := range v.Packages {
		if packageSlug(v.Packages[index].ImportPath) == pathValue || v.Packages[index].ImportPath == pathValue {
			return &v.Packages[index], true
		}
	}
	return nil, false
}

func (p Package) Slug() string {
	return packageSlug(p.ImportPath)
}

func (p Package) Symbol(name string) (kind string, title string, doc string, decl string, ok bool) {
	for _, value := range p.Constants {
		if contains(value.Names, name) {
			return "constant", name, value.Doc, value.Decl, true
		}
	}
	for _, value := range p.Variables {
		if contains(value.Names, name) {
			return "variable", name, value.Doc, value.Decl, true
		}
	}
	for _, fn := range p.Functions {
		if fn.Name == name {
			return "function", name, fn.Doc, fn.Decl, true
		}
	}
	for _, typ := range p.Types {
		if typ.Name == name {
			return "type", name, typ.Doc, typ.Decl, true
		}
		for _, method := range typ.Methods {
			methodName := typ.Name + "." + method.Name
			if method.Name == name || methodName == name {
				return "method", methodName, method.Doc, method.Decl, true
			}
		}
	}
	return "", "", "", "", false
}

func (i *Index) Search(version, query string) []SearchResult {
	query = strings.ToLower(strings.TrimSpace(query))
	if i == nil || query == "" {
		return nil
	}
	versions := i.Versions
	if version != "" {
		if selected, ok := i.Version(version); ok {
			versions = []Version{*selected}
		}
	}
	var results []SearchResult
	for _, version := range versions {
		for _, pkg := range version.Packages {
			packageURL := "/packages/" + version.Version + "/" + pkg.Slug()
			if matches(query, pkg.ImportPath, pkg.Name, pkg.Synopsis, pkg.Doc) {
				results = append(results, SearchResult{
					Version:     version.Version,
					PackagePath: pkg.ImportPath,
					PackageName: pkg.Name,
					Kind:        "package",
					Name:        pkg.ImportPath,
					URL:         packageURL,
					Summary:     firstNonEmpty(pkg.Synopsis, pkg.Doc),
				})
			}
			for _, fn := range pkg.Functions {
				if matches(query, fn.Name, fn.Doc, fn.Decl) {
					results = append(results, SearchResult{
						Version:     version.Version,
						PackagePath: pkg.ImportPath,
						PackageName: pkg.Name,
						Kind:        "function",
						Name:        fn.Name,
						URL:         packageURL + "#" + fn.Name,
						Summary:     firstNonEmpty(fn.Doc, fn.Decl),
					})
				}
			}
			for _, typ := range pkg.Types {
				if matches(query, typ.Name, typ.Doc, typ.Decl) {
					results = append(results, SearchResult{
						Version:     version.Version,
						PackagePath: pkg.ImportPath,
						PackageName: pkg.Name,
						Kind:        "type",
						Name:        typ.Name,
						URL:         packageURL + "#" + typ.Name,
						Summary:     firstNonEmpty(typ.Doc, typ.Decl),
					})
				}
				for _, method := range typ.Methods {
					methodName := typ.Name + "." + method.Name
					if matches(query, methodName, method.Doc, method.Decl) {
						results = append(results, SearchResult{
							Version:     version.Version,
							PackagePath: pkg.ImportPath,
							PackageName: pkg.Name,
							Kind:        "method",
							Name:        methodName,
							URL:         packageURL + "#" + methodName,
							Summary:     firstNonEmpty(method.Doc, method.Decl),
						})
					}
				}
			}
		}
	}
	if len(results) > 30 {
		return results[:30]
	}
	return results
}

func packageSlug(importPath string) string {
	slug := strings.TrimPrefix(importPath, "golazy.dev/")
	if slug == "golazy.dev" {
		return "golazy"
	}
	return strings.ReplaceAll(slug, "/", ".")
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func matches(query string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
