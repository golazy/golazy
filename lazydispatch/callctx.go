package lazydispatch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazysupport"
)

type callctx struct {
	actx         *actionctx
	mi           methodInfo
	instance     reflect.Value
	w            http.ResponseWriter
	r            *http.Request
	err          error
	stringParams []string
}

func (cctx *callctx) fn() reflect.Value {
	return cctx.instance.Method(cctx.mi.method)
}
func (cctx *callctx) fnT() reflect.Type {
	return cctx.fn().Type()
}
func (cctx *callctx) fnIn(i int) reflect.Type {
	return cctx.fnT().In(i)
}
func (cctx *callctx) NumIn() int {
	return cctx.fnT().NumIn()
}

func call(cctx callctx) ([]reflect.Value, error) {
	nInputs := cctx.NumIn()
	inputs := make([]reflect.Value, nInputs)
	for n := 0; n < nInputs; n++ {
		in, err := findInput(&cctx, cctx.fnIn(n))
		if err != nil {
			return []reflect.Value{}, err
		}
		inputs[n] = in
	}

	return cctx.fn().Call(inputs), nil
}

func processActionOutput(cctx *callctx, outs []reflect.Value, w http.ResponseWriter) error {
	body := []byte{}
	var reader io.Reader
	var err error
	var status int
	var header *http.Header
	for _, out := range outs {
		switch v := out.Interface().(type) {
		case nil:
			continue
		case error:
			err = v
		case string:
			body = []byte(v)
		case int:
			status = v
		case int8:
			status = int(v)
		case int16:
			status = int(v)
		case int32:
			status = int(v)
		case int64:
			status = int(v)
		case uint:
			status = int(v)
		case uint8:
			status = int(v)
		case uint16:
			status = int(v)
		case uint32:
			status = int(v)
		case uint64:
			status = int(v)
		case []byte:
			body = v
		case http.Header:
			header = &v
		case io.Reader:
			reader = v
		default:
			panic(fmt.Sprintf("unknown output type %s", out.Type().Name()))
		}
	}
	if header != nil {
		for k, v := range *header {
			for _, v := range v {
				w.Header().Add(k, v)
			}
		}
	}
	if err != nil {
		callErrorHandler(*cctx, err)
		return errStop
		// err = fmt.Errorf("error while calling %s#%s: %w", cctx.actx.t.String(), cctx.mi.name, err)
		// http.Error(w, err.Error(), http.StatusInternalServerError)
		// panic(err)
		// return
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	if len(body) != 0 {
		w.Write(body)
	} else {
		if reader != nil {
			io.Copy(w, reader)
		}
	}
	return nil
}

var (
	tHTTPRequest        = reflect.TypeFor[*http.Request]()
	tHTTPResponseWriter = reflect.TypeFor[http.ResponseWriter]()
	tContextContext     = reflect.TypeFor[context.Context]()
	tError              = reflect.TypeFor[error]()
	tString             = reflect.TypeFor[string]()
)

func findInput(ctx *callctx, t reflect.Type) (reflect.Value, error) {
	name := t.String()

	// Try to find a generator
	for gName, generator := range ctx.actx.generators {
		if gName == name {
			var val reflect.Value
			var err error
			lazysupport.MeasureText(" ├╴"+generator.name, func() error {
				val, err = callGenerator(ctx.actx, generator, ctx.instance, ctx.w, ctx.r)
				if err != nil {
					return err
				}
				return nil
			})
			return val, err
		}
	}

	// Or get it from the context
	out := ctx.r.Context().Value(t)
	if out != nil {
		return reflect.ValueOf(out), nil
	}
	// TODO: Check why values from lazycontext do not work
	//	out = ctx.r.Context().Value(lazycontext.KeyForType(t.Interface()))
	//	if out != nil {
	//		return reflect.ValueOf(out), nil
	//	}

	// Check for basic types
	switch t {
	case tHTTPRequest:
		return reflect.ValueOf(ctx.r), nil
	case tHTTPResponseWriter:
		return reflect.ValueOf(ctx.w), nil
	case tContextContext:
		return reflect.ValueOf(ctx.r.Context()), nil
	case tError:
		return reflect.ValueOf(ctx.err), nil
	case tString:
		if ctx.stringParams == nil {
			r := ctx.r.Context().Value(reflect.TypeFor[*Route]())
			if r == nil {
				return reflect.Value{}, errors.New("string param can't be filled as there is no *Route in the context")
			}
			route := r.(*Route)
			ctx.stringParams = extractParam(ctx.r.URL.Path, route.Path)
		}
		if len(ctx.stringParams) == 0 {
			return reflect.Value{}, fmt.Errorf("method %s#%s asked for more params than available", ctx.actx.t.String(), ctx.mi.name)
		}
		v := reflect.ValueOf(ctx.stringParams[0])
		ctx.stringParams = ctx.stringParams[1:]
		return v, nil
	}

	return reflect.Value{}, fmt.Errorf("parameter %s needed by method %s#%s not found", name, ctx.actx.t.String(), ctx.mi.name)
}

func callErrorHandler(cctx callctx, err error) {
	if cctx.actx.ErrorHandler == nil {
		panic(err)
	}
	// Ensure we don't call the error handler twice
	if cctx.err != nil {
		panic(fmt.Errorf("double panic: %w", cctx.err))
	}
	cctx.err = err
	cctx.mi = *cctx.actx.ErrorHandler
	_, err = call(cctx)
	if err != nil {
		panic(err)
	}
}

func extractParam(url, path string) []string {

	out := []string{}
	tmplComp := strings.Split(path, "/")
	urlComp := strings.Split(url, "/")
	for i, c := range tmplComp {
		if strings.HasPrefix(c, ":") {
			out = append(out, urlComp[i])
		}
	}
	//reverse it
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
