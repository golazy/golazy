package args

import (
	"fmt"
	"reflect"
)

type Fn2 struct {
	Fn   reflect.Value
	Ins  []string
	Outs []string
}

func NewFn2(v any) *Fn2 {

	//t := v2.Type()
	t := reflect.TypeOf(v)

	if t.Kind() != reflect.Func {
		panic(fmt.Sprintf("%q is not a function", t.Name()))
	}

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
	return &Fn2{
		Fn:   fn,
		Ins:  ins,
		Outs: outs,
	}

}
