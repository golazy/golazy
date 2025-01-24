package lazydispatch

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"sync"

	"golazy.dev/lazysupport"
)

type actionctx struct {
	t            reflect.Type
	befores      []methodInfo
	afters       []methodInfo
	ErrorHandler *methodInfo
	generators   map[string]methodInfo
	action       methodInfo
	pool         sync.Pool
	tt           reflect.Type
	vv           reflect.Value
	ctxfn        []func(ctx context.Context, r *http.Request) context.Context
	// TODO: fill thoose and pass them to the action somehow
}

func (actx *actionctx) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	/*
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("ERROR:", err)
				if err, ok := err.(error); ok {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				} else {
					http.Error(w, fmt.Sprintf("%+v", err), http.StatusInternalServerError)
				}
				panic(err)
			}
		}()
	*/

	lazysupport.MeasureText(" └DONE:", func() error {
		fmt.Printf("\n%s %s => %s#%s\n", r.Method, r.URL.Path, actx.tt.String(), actx.action.name)

		// If provided, allow the caller to ForAction to modify the request context
		if len(actx.ctxfn) > 0 {
			r = r.WithContext(actx.ctxfn[0](r.Context(), r))
		}

		// Instanciate
		instance := actx.pool.Get().(*reflect.Value)
		defer func() {
			// TODO: See if the request was hijacked
			actx.pool.Put(instance)
		}()
		// Copy initial values
		instance.Elem().Set(actx.vv)

		// Call before filters
		for _, before := range actx.befores {
			err = lazysupport.MeasureText(" ├╴"+before.name, func() error {
				return actx.callFilter(before, *instance, w, r)
			})
			if err == errStop {
				return errStop
			}
			if err != nil {
				panic(err)
			}
		}

		// Call action
		var outs = []reflect.Value{}

		err := lazysupport.MeasureText(" ├╴"+actx.action.name, func() (err error) {
			cctx := callctx{actx, actx.action, *instance, w, r, nil, nil}
			defer func() {
				p := recover()
				if p != nil {
					pa := lazysupport.NewPanic(p, debug.Stack(), 1)
					callPanicHandler(cctx, pa)
					err = errStop
					return
				}
			}()
			outs, err = call(cctx)
			// If any gen returns an error, we stop the execution
			if err == errStop {
				return err
			}
			if err != nil {
				panic(err)
			}
			return processActionOutput(&callctx{actx, actx.action, *instance, w, r, nil, nil}, outs, w)
		})
		if err == errStop {
			return nil
		}
		if err != nil {
			panic(err)
		}

		// Call after filters
		for _, after := range actx.afters {
			err = lazysupport.MeasureText(" ├╴"+after.name, func() error {
				return actx.callFilter(after, *instance, w, r)
			})
			if err == errStop {
				return nil
			}
			if err != nil {
				panic(err)
			}
		}
		return nil
	})

}
func (actx *actionctx) callFilter(mi methodInfo, instance reflect.Value, w http.ResponseWriter, r *http.Request) (err error) {
	cctx := callctx{actx, mi, instance, w, r, nil, nil}

	defer func() {
		if p := recover(); p != nil {
			pa := lazysupport.NewPanic(p, debug.Stack(), 1)
			callPanicHandler(cctx, pa)
			err = errStop
			return
		}
	}()
	outs, err := call(cctx)
	if err != nil {
		return err
	}
	if len(outs) == 1 && !outs[0].IsNil() {
		err, ok := outs[0].Interface().(error)
		if !ok {
			err = fmt.Errorf("%s returned a non error(%s) value as first return value", mi.name, outs[0].Type().String())
			panic(err)
		}
		if err != nil {
			callErrorHandler(callctx{actx, mi, instance, w, r, nil, nil}, err)
			return errStop
		}
	}
	return nil
}

func callGenerator(actx *actionctx, mi methodInfo, instance reflect.Value, w http.ResponseWriter, r *http.Request) (value reflect.Value, err error) {
	// TODO: Check that out is not included in the input (infinit loop)
	cctx := callctx{actx, mi, instance, w, r, nil, nil}
	var outs []reflect.Value

	defer func() {
		p := recover()
		if p != nil {
			pa := lazysupport.NewPanic(p, debug.Stack(), 1)
			callPanicHandler(cctx, pa)
			err = errStop
			return
		}
	}()
	outs, err = call(cctx)
	if err != nil {
		panic(err)
	}
	if len(outs) == 2 {
		v := outs[1].Interface()
		if v != nil {
			err, ok := v.(error)
			if !ok {
				panic(fmt.Errorf("%s returned a non error(%s) value as second return value", mi.name, outs[1].Type().String()))
			}
			if err == errStop {
				return outs[0], err
			}
			if err != nil {
				callErrorHandler(cctx, err)
				return reflect.Value{}, errStop
			}
		}
	}
	return outs[0], nil
}

func callPanicHandler(cctx callctx, err error) {
	if cctx.actx.ErrorHandler == nil {
		panic(err)
	}
	// Ensure we don't call the error handler twice
	if cctx.err != nil {
		panic(fmt.Errorf("double panic: %w", cctx.err))
	}
	cctx.err = err
	_, err = call(cctx)
	if err != nil {
		panic(err)
	}
}
