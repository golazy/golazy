package lazysupport

import "reflect"

func NameOfWithPackage(obj any) string {
	if obj == nil {
		panic("NameOf called with nil")
	}

	t := reflect.TypeOf(obj)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:

	case reflect.Slice, reflect.Map, reflect.Array:
		t = t.Elem()
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		return Pluralize(t.String())

	default:
		panic("NameOf called with non struct, slice or map")
	}

	return t.String()

}

func NameOf(obj any) string {
	if obj == nil {
		panic("NameOf called with nil")
	}

	t := reflect.TypeOf(obj)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:

	case reflect.Slice, reflect.Map, reflect.Array:
		t = t.Elem()
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		return Pluralize(t.Name())

	default:
		panic("NameOf called with non struct, slice or map")
	}

	return t.Name()

}
