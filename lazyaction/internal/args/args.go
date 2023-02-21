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
