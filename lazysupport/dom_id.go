package lazysupport

import (
	"fmt"
	"reflect"
)

func DomID(s any) (string, error) {

	if d, ok := s.(DomIDer); ok {
		return d.DomID(), nil
	}
	if s == nil {
		return "", fmt.Errorf("cannot get ID from nil")
	}

	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("cannot get ID from %T", s)
	}

	name := Underscorize(t.Name())
	var err error

	id, err := IDFor(s)
	if err != nil {
		return "", err
	}
	return name + "_" + id, nil

}

type DomIDer interface {
	DomID() string
}

type ToParamer interface {
	ToParam() string
}

func IDFor(s any) (string, error) {
	if s == nil {
		return "", fmt.Errorf("cannot get ID from nil")
	}
	if s, ok := s.(ToParamer); ok {
		return s.ToParam(), nil
	}
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return fmt.Sprint(s), nil
	case reflect.Struct:
	default:
		return "", fmt.Errorf("cannot get ID from %T", s)
	}

	// ID field
	f := v.FieldByName("ID")
	if f.IsValid() {
		i := fmt.Sprint(f.Interface())
		return i, nil
	}

	// Id field
	f = v.FieldByName("Id")
	if f.IsValid() {
		i := fmt.Sprint(f.Interface())
		return i, nil
	}

	// ID method
	f = v.MethodByName("ID")
	if f.IsValid() {
		// Check receiving arguemnts
		mt, _ := t.MethodByName("ID")
		if mt.Type.NumIn() == 0 || mt.Type.NumOut() == 1 {
			// Check returning arguments
			i := fmt.Sprint(f.Call([]reflect.Value{})[0].Interface())
			return i, nil
		}
	}
	// Id method
	f = v.MethodByName("Id")
	if f.IsValid() {
		// Check receiving arguemnts
		mt, _ := t.MethodByName("Id")
		if mt.Type.NumIn() == 0 || mt.Type.NumOut() == 1 {
			// Check returning arguments
			i := fmt.Sprint(f.Call([]reflect.Value{})[0].Interface())
			return i, nil
		}
	}

	f = v.MethodByName("Id")
	if f.IsValid() {
		i := fmt.Sprint(f.Call([]reflect.Value{})[0].Interface())
		return i, nil
	}

	return "", fmt.Errorf("cannot get ID from %T", s)
}
