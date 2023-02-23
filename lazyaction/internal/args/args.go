package args

import (
	"fmt"
	"reflect"
)

var (
	ErrNilAction     = fmt.Errorf("action is nil")
	ErrNonFuncAction = fmt.Errorf("action is not a function")
)

func ExtractArgs(t reflect.Type) (args, rets []string, err error) {
	if t == nil {
		return nil, nil, ErrNilAction
	}
	if t.Kind() != reflect.Func {
		return nil, nil, ErrNonFuncAction
	}

	args = make([]string, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		args[i] = t.In(i).String()
	}

	rets = make([]string, t.NumOut())

	for i := 0; i < t.NumOut(); i++ {
		rets[i] = t.Out(i).String()
	}

	return
}

type Fn struct {
	Fn   reflect.Value
	Ins  []string
	Outs []string
}

type InputSet struct {
	Generators map[string][]Gen
	Values     map[string][]any
}

type Gen Fn

func (g Gen) Call(inputs InputSet) (o []reflect.Value, err error) {
	rets, err := Fn(g).Call(inputs)
	if err != nil {
		return rets, err
	}
	if len(g.Outs) == 2 {
		if rets[1].IsNil() {
			return rets, nil
		}
		return rets, rets[1].Interface().(error)
	}
	return rets, nil

}

func NewGen(v any) Gen {
	fn := NewFn(v)
	outs := fn.Outs
	if len(outs) == 0 {
		panic("generator must return at least one value")
	}

	if len(outs) > 2 {
		panic("generator can return at most two values")
	}
	if len(outs) == 2 && outs[1] != "error" {
		panic("second return value must be error")
	}
	return Gen(*fn)
}

type ErrArgumentNotFound string

func (e ErrArgumentNotFound) Error() string {
	return fmt.Sprintf("argument %s not found", string(e))
}

func (f Fn) Call(inputs InputSet) (o []reflect.Value, err error) {
	// Test for cycles
	ins := make([]reflect.Value, len(f.Ins))

	typeCount := make(map[string]int)

	for i, inputType := range f.Ins {
		val, ok := inputs.Values[inputType]
		if ok {
			if typeCount[inputType] >= len(val) {
				err = fmt.Errorf("missing values for type %s", inputType)
				return
			}
			rv, ok := val[typeCount[inputType]].(reflect.Value)
			if ok {
				ins[i] = rv
			} else {
				ins[i] = reflect.ValueOf(val[typeCount[inputType]])
			}

			typeCount[inputType]++
			continue
		}

		// Find the generator
		gens, ok := inputs.Generators[inputType]
		if !ok {
			err = ErrArgumentNotFound(inputType)
			return
		}

		if typeCount[inputType] >= len(gens) {
			err = fmt.Errorf("missing generators for type %s", inputType)
			return
		}
		gen := gens[typeCount[inputType]]
		typeCount[inputType]++

		genOut := []reflect.Value{}
		genOut, err = gen.Call(inputs)
		if err != nil {
			return
		}
		if len(gen.Outs) == 2 && gen.Outs[1] != "error" {
			err = fmt.Errorf("generator %s returned second value that is not error", inputType)
			return
		}
		if len(gen.Outs) == 2 {
			if !genOut[1].IsNil() {
				err = genOut[1].Interface().(error)
				return
			}
		}
		ins[i] = genOut[0]
	}

	return f.Fn.Call(ins), nil
}

type OutputSet struct {
	Types  []string
	Values []reflect.Value
}

func NewFn(v any) *Fn {
	fn, ok := v.(reflect.Value)
	if !ok {
		fn = reflect.ValueOf(v)
	}
	if fn.Kind() != reflect.Func {
		panic("not a function")
	}
	ins, outs, err := ExtractArgs(fn.Type())
	if err != nil {
		panic(err)
	}
	return &Fn{
		Fn:   fn,
		Ins:  ins,
		Outs: outs,
	}
}

func OutsToInputs(outs []reflect.Value) InputSet {
	values := make(map[string][]any)

	for _, out := range outs {
		values[out.Type().String()] = append(values[out.Type().String()], out.Interface())
	}
	return InputSet{
		Values: values,
	}
}

func (is InputSet) Merge(is2 InputSet) InputSet {
	newIs := InputSet{
		Values:     make(map[string][]any),
		Generators: make(map[string][]Gen),
	}
	// Copy original values
	for k, v := range is.Values {
		newIs.Values[k] = append(newIs.Values[k], v...)
	}
	for k, v := range is.Generators {
		newIs.Generators[k] = append(newIs.Generators[k], v...)
	}

	// Append new ones
	for k, v := range is2.Values {
		is.Values[k] = append(is.Values[k], v...)
	}
	for k, v := range is2.Generators {
		is.Generators[k] = append(is.Generators[k], v...)
	}
	return is
}

/*

func EachMethod(v any) []*Fn {
	fns := map[string]*Fn{}
	val := reflect.ValueOf(v)
	for i := 0; i < val.NumMethod(); i++ {
		n := val.Method(i).Type().Name()
		fns[n] = NewFn(val.Method(i))
	}


	// Inspect all the embebed types
	for i := 0; i < val.NumField(); i++ {
		f := val.Field(i)
		if f.Kind() != reflect.Struct {
			continue
		}
		if f.Anonymous {
			for _, fn := range EachMethod(f.Interface()) {
				name := fn.Fn.Type().Name()
				if _, ok := fns[name]; !ok {
					fn[fn.Fn.Type().Name()] = fn
				if _, ok :=
				}
				fns[fn.Fn.Type().Name()] = fn
			}
		}



	return fns
}

*/
