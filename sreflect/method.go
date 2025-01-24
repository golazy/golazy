package sreflect

import (
	"fmt"
	"reflect"
	"slices"
)

type Method struct {
	Struct          *Type
	StructNumMethod int
	Name            string
	Type            reflect.Type
	Field           reflect.Method
	Inputs          []reflect.Type
	Outputs         []reflect.Type
}

func (m *Method) String() string {
	return m.Struct.String() + "." + m.Name
}

type Resolver func(reflect.Type) (reflect.Value, error)

func (m *Method) Call(instance reflect.Value, resolver Resolver) ([]reflect.Value, error) {

	str, err := m.getInstance(instance)
	instanceName := str.Type().String()
	methodName := m.String()
	println(instanceName, methodName)
	if err != nil {
		return nil, err
	}

	// Fill the inputs
	inputs := make([]reflect.Value, len(m.Inputs))

	for i, in := range m.Inputs {
		input, err := resolver(in)
		if err != nil {
			return nil, fmt.Errorf("can't call %s. Argument \"%s\": %w", m.String(), in.String(), err)
		}
		inputs[i] = input
	}
	method := str.Method(m.StructNumMethod)
	mType := method.Type().String()
	print(mType)
	out := method.Call(inputs)

	return out, nil
}

type ErrIsNil struct {
	v reflect.Type
}

func (e ErrIsNil) Error() string {
	return fmt.Sprintf("%s is nil", e.v.String())
}

type ErrNotAStruct struct {
	v reflect.Value
}

func (e ErrNotAStruct) Error() string {
	typeName := ""
	t := e.v.Type()
	name := t.String()
	if t.Kind() == reflect.Ptr {
		typeName = "pointer to "
		t = t.Elem()
	}
	typeName += t.Kind().String()
	return fmt.Sprintf("%s (%s) is not a struct", name, typeName)
}

func (m *Method) getInstance(instance reflect.Value) (reflect.Value, error) {
	t := instance.Type()
	if t.Kind() != reflect.Ptr && (t.Kind() == reflect.Ptr && t.Elem().Kind() != reflect.Struct) {
		return reflect.Value{}, ErrNotAStruct{instance}
	}
	if t.Kind() == reflect.Ptr && instance.IsNil() {
		return reflect.Value{}, ErrIsNil{instance.Type()}
	}
	if t.Kind() == reflect.Ptr {
		instance = instance.Elem()
	}

	// The instance is for the Struct top level parent.
	// We need to collect the numfield for each parent
	// and get the value of each one, returning ErrFieldIsNil
	// if any of them is a nil pointer

	fields := []int{}
	for p := m.Struct; p.parent != nil; p = p.parent {
		fields = append(fields, p.parentNumField)
	}
	slices.Reverse(fields)
	for _, f := range fields {

		instance = instance.Field(f)
		name := instance.Type().String()
		print(name)
		if instance.Kind() == reflect.Ptr && instance.IsNil() {
			return reflect.Value{}, ErrIsNil{}
		}
	}
	if instance.Kind() == reflect.Ptr {
		return instance, nil
	}
	return instance.Addr(), nil

}

func newMethod(s *Type, field reflect.Method, i int) Method {
	ins := field.Type.NumIn()
	outs := field.Type.NumOut()
	m := Method{
		Struct:          s,
		Name:            field.Name,
		StructNumMethod: i,
		Type:            field.Type,
		Field:           field,
		Inputs:          make([]reflect.Type, ins-1),
		Outputs:         make([]reflect.Type, outs),
	}
	for i := 1; i < ins; i++ {
		m.Inputs[i-1] = field.Type.In(i)
	}
	for i := 0; i < outs; i++ {
		m.Outputs[i] = field.Type.Out(i)
	}
	return m
}
