package lazyschema

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// Path is a schema field path made of generated field aliases and optional
// slice indices.
type Path []string

// NameOption configures generated field names and ids.
type NameOption func(*nameOptions)

type nameOptions struct {
	prefix string
}

// WithPrefix prefixes generated field ids with a model or form key.
func WithPrefix(prefix string) NameOption {
	return func(options *nameOptions) {
		options.prefix = strings.Trim(prefix, "_")
	}
}

// ModelName returns the default lower-camel key for a model value or type.
func ModelName(model any) (string, error) {
	t, err := structType(model)
	if err != nil {
		return "", err
	}
	return ModelNameForType(t)
}

// ModelNameForType returns the default lower-camel key for a struct type.
func ModelNameForType(t reflect.Type) (string, error) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("lazyschema: model type must be a struct or pointer to struct")
	}
	return defaultAlias(t.Name()), nil
}

// PathFor returns the generated schema path for a Go field path such as
// "Phone.Label" or "Phones.0.Number".
func PathFor(model any, field string) (Path, error) {
	t, err := structType(model)
	if err != nil {
		return nil, err
	}
	return pathForType(t, strings.Split(strings.Trim(field, "."), "."))
}

// FieldName returns the form field name for a schema path.
func FieldName(path Path, _ ...NameOption) (string, error) {
	return joinPath(path)
}

// FieldID returns the DOM id for a schema path.
func FieldID(path Path, opts ...NameOption) (string, error) {
	var options nameOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	name, err := joinPath(path)
	if err != nil {
		return "", err
	}
	if options.prefix == "" {
		return name, nil
	}
	return options.prefix + "_" + name, nil
}

// FieldNameFor returns the generated form field name for a Go field path.
func FieldNameFor(model any, field string, opts ...NameOption) (string, error) {
	path, err := PathFor(model, field)
	if err != nil {
		return "", err
	}
	return FieldName(path, opts...)
}

// FieldIDFor returns the generated DOM id for a Go field path.
func FieldIDFor(model any, field string, opts ...NameOption) (string, error) {
	path, err := PathFor(model, field)
	if err != nil {
		return "", err
	}
	return FieldID(path, opts...)
}

func pathForType(t reflect.Type, parts []string) (Path, error) {
	if len(parts) == 0 || parts[0] == "" {
		return nil, fmt.Errorf("lazyschema: field path is required")
	}

	cache := newCache()
	var out Path
	for index := 0; index < len(parts); index++ {
		if t.Kind() != reflect.Struct {
			return nil, errInvalidPath
		}
		info := cache.get(t)
		if info.err != nil {
			return nil, info.err
		}
		part := parts[index]
		field := info.fieldByName(part)
		if field == nil {
			field = info.get(part)
		}
		if field == nil {
			return nil, UnknownKeyError{Key: part}
		}
		out = append(out, field.alias)
		t = indirectType(field.typ)
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = indirectType(t.Elem())
			if index+1 < len(parts) {
				if _, err := strconv.Atoi(parts[index+1]); err == nil {
					index++
					out = append(out, parts[index])
				}
			}
		}
	}
	return out, nil
}

func (i *structInfo) fieldByName(name string) *fieldInfo {
	for _, field := range i.fields {
		if field.name == name {
			return field
		}
	}
	return nil
}

func joinPath(path Path) (string, error) {
	if len(path) == 0 {
		return "", fmt.Errorf("lazyschema: field path is required")
	}
	for _, part := range path {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("lazyschema: field path contains an empty segment")
		}
	}
	return strings.Join(path, "_"), nil
}

func structType(model any) (reflect.Type, error) {
	if model == nil {
		return nil, fmt.Errorf("lazyschema: model is nil")
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("lazyschema: model must be a struct or pointer to struct")
	}
	return t, nil
}

func defaultAlias(name string) string {
	if name == "" {
		return name
	}
	runes := []rune(name)
	upper := 0
	for upper < len(runes) && unicode.IsUpper(runes[upper]) {
		upper++
	}
	if upper == 0 {
		return name
	}
	if upper == len(runes) {
		return strings.ToLower(name)
	}
	if upper > 1 && unicode.IsLower(runes[upper]) {
		upper--
	}
	var builder strings.Builder
	for i, r := range runes {
		if i < upper {
			builder.WriteRune(unicode.ToLower(r))
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}
