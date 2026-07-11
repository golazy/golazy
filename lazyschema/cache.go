// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.gorilla file.

package lazyschema

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
)

var errInvalidPath = errors.New("lazyschema: invalid path")

// newCache returns a new cache.
func newCache() *cache {
	c := cache{
		m:       make(map[reflect.Type]*structInfo),
		regconv: make(map[reflect.Type]Converter),
		tag:     "schema",
	}
	return &c
}

// cache caches meta-data about a struct.
type cache struct {
	l       sync.RWMutex
	m       map[reflect.Type]*structInfo
	regconv map[reflect.Type]Converter
	tag     string
}

// registerConverter registers a converter function for a custom type.
func (c *cache) registerConverter(value any, converterFunc Converter) {
	c.regconv[reflect.TypeOf(value)] = converterFunc
}

// parsePath parses a path in dotted notation verifying that it is a valid
// path to a struct field.
//
// It returns "path parts" which contain indices to fields to be used by
// reflect.Value.FieldByString(). Multiple parts are required for slices of
// structs.
func (c *cache) parsePath(p string, t reflect.Type) ([]pathPart, error) {
	if strings.Contains(p, ".") {
		return c.parsePathParts(strings.Split(p, "."), t)
	}
	return c.parsePathParts(strings.Split(p, "_"), t)
}

func (c *cache) parsePathParts(keys []string, t reflect.Type) ([]pathPart, error) {
	var struc *structInfo
	var field *fieldInfo
	var index64 int64
	var err error
	parts := make([]pathPart, 0)
	path := make([]string, 0)
	for i := 0; i < len(keys); {
		if t.Kind() != reflect.Struct {
			return nil, errInvalidPath
		}
		if struc = c.get(t); struc == nil {
			return nil, errInvalidPath
		}
		if struc.err != nil {
			return nil, struc.err
		}
		next := i + 1
		if field = struc.get(keys[i]); field == nil {
			field, next = struc.match(keys, i)
		}
		if field == nil {
			return nil, errInvalidPath
		}
		i = next
		// Valid field. Append index.
		path = append(path, field.name)
		if field.isSliceOfStructs && (!field.unmarshalerInfo.IsValid || (field.unmarshalerInfo.IsValid && field.unmarshalerInfo.IsSliceElement)) {
			// Parse a special case: slices of structs.
			// The next key must be the slice index.
			//
			// Now that struct can implements TextUnmarshaler interface,
			// we don't need to force the struct's fields to appear in the path.
			// So checking i+2 is not necessary anymore.
			if i >= len(keys) {
				return nil, errInvalidPath
			}
			if index64, err = strconv.ParseInt(keys[i], 10, 0); err != nil {
				return nil, errInvalidPath
			}
			i++
			parts = append(parts, pathPart{
				path:  path,
				field: field,
				index: int(index64),
			})
			path = make([]string, 0)

			// Get the next struct type, dropping ptrs.
			if field.typ.Kind() == reflect.Pointer {
				t = field.typ.Elem()
			} else {
				t = field.typ
			}
			if t.Kind() == reflect.Slice {
				t = t.Elem()
				if t.Kind() == reflect.Pointer {
					t = t.Elem()
				}
			}
		} else if field.typ.Kind() == reflect.Pointer {
			t = field.typ.Elem()
		} else {
			t = field.typ
		}
	}
	// Add the remaining.
	parts = append(parts, pathPart{
		path:  path,
		field: field,
		index: -1,
	})
	return parts, nil
}

// get returns a cached structInfo, creating it if necessary.
func (c *cache) get(t reflect.Type) *structInfo {
	c.l.RLock()
	info := c.m[t]
	c.l.RUnlock()
	if info == nil {
		info = c.create(t, "")
		c.l.Lock()
		c.m[t] = info
		c.l.Unlock()
	}
	return info
}

// create creates a structInfo with meta-data about a struct.
func (c *cache) create(t reflect.Type, parentAlias string) *structInfo {
	info := &structInfo{}
	var anonymousInfos []*structInfo
	for field := range t.Fields() {
		if f := c.createField(field, parentAlias); f != nil {
			info.fields = append(info.fields, f)
			if ft := indirectType(f.typ); ft.Kind() == reflect.Struct && f.isAnonymous {
				anonymousInfos = append(anonymousInfos, c.create(ft, f.canonicalAlias))
			}
		}
	}
	for i, a := range anonymousInfos {
		others := []*structInfo{info}
		others = append(others, anonymousInfos[:i]...)
		others = append(others, anonymousInfos[i+1:]...)
		for _, f := range a.fields {
			if !containsAlias(others, f.alias) {
				info.fields = append(info.fields, f)
			}
		}
	}
	info.err = info.checkConflicts(t)
	return info
}

// createField creates a fieldInfo for the given field.
func (c *cache) createField(field reflect.StructField, parentAlias string) *fieldInfo {
	alias, options := fieldAlias(field, c.tag)
	if alias == "-" {
		// Ignore this field.
		return nil
	}
	canonicalAlias := alias
	if parentAlias != "" {
		canonicalAlias = parentAlias + "_" + alias
	}
	// Check if the type is supported and don't cache it if not.
	// First let's get the basic type.
	isSlice, isStruct := false, false
	ft := field.Type
	m := isTextUnmarshaler(reflect.Zero(ft))
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	if isSlice = ft.Kind() == reflect.Slice; isSlice {
		ft = ft.Elem()
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
	}
	if ft.Kind() == reflect.Array {
		ft = ft.Elem()
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
	}
	if isStruct = ft.Kind() == reflect.Struct; !isStruct {
		if c.converter(ft) == nil && builtinConverters[ft.Kind()] == nil {
			// Type is not supported.
			return nil
		}
	}

	return &fieldInfo{
		typ:              field.Type,
		name:             field.Name,
		alias:            alias,
		canonicalAlias:   canonicalAlias,
		unmarshalerInfo:  m,
		isSliceOfStructs: isSlice && isStruct,
		isAnonymous:      field.Anonymous,
		isRequired:       options.Contains("required"),
		defaultValue:     options.getDefaultOptionValue(),
	}
}

// converter returns the converter for a type.
func (c *cache) converter(t reflect.Type) Converter {
	return c.regconv[t]
}

// ----------------------------------------------------------------------------

type structInfo struct {
	fields []*fieldInfo
	err    error
}

func (i *structInfo) get(alias string) *fieldInfo {
	for _, field := range i.fields {
		if strings.EqualFold(field.alias, alias) {
			return field
		}
	}
	return nil
}

func (i *structInfo) match(parts []string, start int) (*fieldInfo, int) {
	for end := len(parts); end > start; end-- {
		if field := i.get(strings.Join(parts[start:end], "_")); field != nil {
			return field, end
		}
	}
	return nil, start
}

func (i *structInfo) checkConflicts(t reflect.Type) error {
	aliases := map[string]string{}
	for _, field := range i.fields {
		key := strings.ToLower(field.alias)
		if previous, ok := aliases[key]; ok {
			return NameConflictError{
				Type:   t,
				Key:    field.alias,
				Fields: []string{previous, field.name},
			}
		}
		aliases[key] = field.name
	}
	return nil
}

func containsAlias(infos []*structInfo, alias string) bool {
	for _, info := range infos {
		if info.get(alias) != nil {
			return true
		}
	}
	return false
}

type fieldInfo struct {
	typ reflect.Type
	// name is the field name in the struct.
	name  string
	alias string
	// canonicalAlias is almost the same as the alias, but is prefixed with
	// an embedded struct field alias if this field is
	// promoted from the struct.
	// For instance, if the alias is "N" and this field is an embedded field
	// in a struct "X", canonicalAlias will be "x_n".
	canonicalAlias string
	// unmarshalerInfo contains information regarding the
	// encoding.TextUnmarshaler implementation of the field type.
	unmarshalerInfo unmarshaler
	// isSliceOfStructs indicates if the field type is a slice of structs.
	isSliceOfStructs bool
	// isAnonymous indicates whether the field is embedded in the struct.
	isAnonymous  bool
	isRequired   bool
	defaultValue string
}

func (f *fieldInfo) paths(prefix string) []string {
	if f.alias == f.canonicalAlias {
		return []string{prefix + f.alias}
	}
	return []string{prefix + f.alias, prefix + f.canonicalAlias}
}

type pathPart struct {
	field *fieldInfo
	path  []string // path to the field: walks structs using field names.
	index int      // struct index in slices of structs.
}

// ----------------------------------------------------------------------------

func indirectType(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Pointer {
		return typ.Elem()
	}
	return typ
}

// fieldAlias parses a field tag to get a field alias.
func fieldAlias(field reflect.StructField, tagName string) (alias string, options tagOptions) {
	if tag := field.Tag.Get(tagName); tag != "" {
		alias, options = parseTag(tag)
	}
	if alias == "" {
		alias = defaultAlias(field.Name)
	}
	return alias, options
}

// tagOptions is the string following a comma in a struct field's tag, or
// the empty string. It does not include the leading comma.
type tagOptions []string

// parseTag splits a struct field's url tag into its name and comma-separated
// options.
func parseTag(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	return s[0], s[1:]
}

// Contains checks whether the tagOptions contains the specified option.
func (o tagOptions) Contains(option string) bool {
	return slices.Contains(o, option)
}

func (o tagOptions) getDefaultOptionValue() string {
	for _, s := range o {
		if strings.HasPrefix(s, "default:") {
			return strings.Split(s, ":")[1]
		}
	}

	return ""
}

// NameConflictError reports two exported struct fields with the same generated
// schema key.
type NameConflictError struct {
	Type   reflect.Type
	Key    string
	Fields []string
}

func (e NameConflictError) Error() string {
	return fmt.Sprintf("lazyschema: %s fields %s share generated key %q", e.Type, strings.Join(e.Fields, ", "), e.Key)
}
