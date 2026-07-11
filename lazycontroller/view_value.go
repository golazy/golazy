package lazycontroller

import (
	"context"
	"fmt"
	"reflect"
	"sync"
)

// Valuer resolves request-local view data when a template or controller needs it.
type Valuer interface {
	Value() (any, error)
}

type viewValue struct {
	ctx     context.Context
	loader  func(context.Context) (any, error)
	loadErr error
	async   bool

	once  sync.Once
	done  chan struct{}
	value any
	err   error
}

// SetLater stores a Valuer and starts loading it immediately.
func (b *Base) SetLater(name string, loader any) Valuer {
	value := newViewValue(b.valueContext(), loader, true)
	b.Set(name, value)
	return value
}

// SetWhenNeeded stores a Valuer that loads on its first Value call.
func (b *Base) SetWhenNeeded(name string, loader any) Valuer {
	value := newViewValue(b.valueContext(), loader, false)
	b.Set(name, value)
	return value
}

func (b *Base) valueContext() context.Context {
	if b == nil || b.ctx == nil {
		return context.Background()
	}
	return b.ctx
}

func newViewValue(ctx context.Context, loader any, async bool) *viewValue {
	if ctx == nil {
		ctx = context.Background()
	}
	load, err := normalizeViewValueLoader(loader)
	value := &viewValue{
		ctx:     ctx,
		loader:  load,
		loadErr: err,
		async:   async,
		done:    make(chan struct{}),
	}
	if async {
		value.start()
	}
	return value
}

func (v *viewValue) Value() (any, error) {
	if v == nil {
		return nil, fmt.Errorf("lazycontroller: view value is nil")
	}
	v.once.Do(func() {
		if v.async {
			go v.resolve()
			return
		}
		v.resolve()
	})
	<-v.done
	return v.value, v.err
}

func (v *viewValue) start() {
	v.once.Do(func() {
		go v.resolve()
	})
}

func (v *viewValue) resolve() {
	defer func() {
		if recovered := recover(); recovered != nil {
			v.value = nil
			v.err = fmt.Errorf("lazycontroller: view value panic: %v", recovered)
		}
		close(v.done)
	}()
	if v.loadErr != nil {
		v.err = v.loadErr
		return
	}
	v.value, v.err = v.loader(v.ctx)
}

var (
	errorType   = reflect.TypeFor[error]()
	contextType = reflect.TypeFor[context.Context]()
)

func normalizeViewValueLoader(loader any) (func(context.Context) (any, error), error) {
	if loader == nil {
		return nil, fmt.Errorf("lazycontroller: view value loader is nil")
	}
	value := reflect.ValueOf(loader)
	typ := value.Type()
	if typ.Kind() != reflect.Func {
		return nil, fmt.Errorf("lazycontroller: view value loader must be a function")
	}
	if typ.NumIn() > 1 {
		return nil, fmt.Errorf("lazycontroller: view value loader must accept zero arguments or context.Context")
	}
	if typ.NumIn() == 1 && !contextType.AssignableTo(typ.In(0)) {
		return nil, fmt.Errorf("lazycontroller: view value loader argument must accept context.Context")
	}
	if typ.NumOut() != 2 || !typ.Out(1).Implements(errorType) {
		return nil, fmt.Errorf("lazycontroller: view value loader must return value, error")
	}

	return func(ctx context.Context) (any, error) {
		var args []reflect.Value
		if typ.NumIn() == 1 {
			args = []reflect.Value{reflect.ValueOf(ctx)}
		}
		out := value.Call(args)
		if err := errorValue(out[1]); err != nil {
			return nil, err
		}
		return out[0].Interface(), nil
	}, nil
}

func errorValue(value reflect.Value) error {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil
		}
	}
	return value.Interface().(error)
}
