package reflect

import (
	"fmt"
	"reflect"
	"strings"
)

type Method struct {
	Parents []reflect.Type
	Method  reflect.Method
	Args    []reflect.Type
	Ret     []reflect.Type
}

func (m *Method) String() string {
	path := make([]string, len(m.Parents)+1)
	for i, parent := range m.Parents {
		path[i] = parent.Name()
	}

	return fmt.Sprintf("func (%s) %s()",
		strings.Join(path, "::"),
		m.Method.Name,
	)
}

func RecursiveCall(method string, args ...any) any {
	return "not implemented"
}

func listMethods(t reflect.Type, parents ...reflect.Type) []Method {

	newParents := append(parents, t)

	methods := []Method{}

	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		fmt.Println(m)
		methods = append(methods, Method{
			Parents: newParents,
			Method:  m,
		})

		// TODO: Add arguments
		// TODO: Add returns
	}

	// NumField() have to be called in a struct
	for t.Kind() == reflect.Pointer {
		newMethods := listMethods(t.Elem(), parents...)
		methods = append(methods, newMethods...)
		return methods
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldT := field.Type
		fmt.Printf("=> %+v %+v  %+v\n", t, field, fieldT)
		if !field.Anonymous {
			continue
		}
		if field.Type.Kind() != reflect.Pointer {
		}
		newMethods := listMethods(field.Type, newParents...)
		fmt.Println("#> ", field.Type.Name(), newMethods)
		methods = append(methods, newMethods...)

	}

	return methods
}

func ReflectAbout(obj any) ([]Method, error) {
	return listMethods(reflect.TypeOf(obj)), nil
}
