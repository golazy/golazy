/* package sreflect helps describing go structs its methods, fields and embed fields */
package sreflect

import (
	"reflect"
	"slices"
	"strings"
)

type Type struct {
	t              reflect.Type
	field          reflect.StructField
	parent         *Type
	parentNumField int

	// Structs contains all embed structs
	// Cycles are ignored
	Structs []Type

	// Methods contains all the methods of the structs and the ones of the embed structs
	Methods []Method
}

func (s *Type) String() string {
	parts := []*Type{s}
	for p := s.parent; p != nil; p = p.parent {
		parts = append(parts, p)
	}
	slices.Reverse(parts)
	name := []string{}
	name = append(name, parts[0].Name())
	for _, p := range parts[1:] {
		name = append(name, p.field.Name)
	}
	return strings.Join(name, ".")
}

// Name returns the name of the struct
func (s *Type) Name() string {
	return s.t.String()
}

func Reflect[T any](s T) *Type {
	t := reflect.TypeOf(s)
	if t.Kind() != reflect.Pointer {
		panic("sreflect: expected a pointer to struct")
	}

	if t.Elem().Kind() != reflect.Struct {
		panic("sreflect: expected a pointer to struct")
	}

	v := reflect.ValueOf(s)
	if v.IsNil() {
		panic("sreflect: expected a non-nil pointer to a struct.")
	}

	si := &Type{
		t: t,
	}
	si.init()
	return si
}
func (s *Type) init() {
	s.Structs = s.genStructs()
	s.Methods = s.genMethods()
}

func (s *Type) tPointer() reflect.Type {
	if s.t.Kind() == reflect.Ptr {
		return s.t
	}
	return reflect.PointerTo(s.t)
}

func (s *Type) sType() reflect.Type {
	if s.t.Kind() == reflect.Ptr {
		return s.t.Elem()
	}
	return s.t
}

func (s *Type) genStructs() []Type {
	// TODO: support interfaces
	structs := []Type{}

fieldLoop:
	for i := 0; i < s.sType().NumField(); i++ {
		field := s.sType().Field(i)
		if !field.Anonymous {
			continue
		}
		if !isStructOrPointerToStruct(field.Type) {
			continue
		}
		// Dicard pointers to existing structures
		if field.Type == s.tPointer() || field.Type == s.sType() {
			continue
		}

		si := &Type{
			t:              field.Type,
			parentNumField: i,
			field:          field,
			parent:         s,
		}

		// Check for cycles
		for p := s.parent; p != nil; p = p.parent {
			if p.tPointer() == si.tPointer() {
				continue fieldLoop
			}
		}
		//println(si.String())

		si.init()
		structs = append(structs, *si)

	}

	return structs
}

func isStructOrPointerToStruct(t reflect.Type) bool {
	if t.Kind() == reflect.Struct {
		return true
	}

	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return true
	}

	return false
}

func (s *Type) genMethods() []Method {

	methods := []Method{}

	for i := 0; i < s.tPointer().NumMethod(); i++ {
		field := s.tPointer().Method(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		m := newMethod(s, field, i)
		methods = append(methods, m)
	}

	return methods
}

func (s *Type) AllStructs() []*Type {
	structs := []*Type{s}
	for _, s := range s.Structs {
		structs = append(structs, s.AllStructs()...)
	}
	return structs
}

func (s *Type) AllMethods() []*Method {
	methods := []*Method{}
	for _, m := range s.Methods {
		methods = append(methods, &m)
	}

	for _, s := range s.Structs {
		methods = append(methods, s.AllMethods()...)
	}
	return methods

}
