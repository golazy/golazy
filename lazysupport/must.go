package lazysupport

import (
	"fmt"
	"reflect"
)

func Must[T any](obj T) T {
	t := reflect.TypeOf(obj)
	if t.Kind() != reflect.Ptr {
		return obj
	}
	v := reflect.ValueOf(obj)
	if v.IsNil() {
		panic(fmt.Errorf("expected %s to not be nil", t.Name()))
	}
	return obj
}
