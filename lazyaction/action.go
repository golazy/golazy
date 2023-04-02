package lazyaction

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/actionrecorder"
	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyassets"
)

type Action struct {
	Dispatcher *Dispatcher
	ins        []string
	outs       []string
	Assets     *lazyassets.Assets
	methodI    int
	fn         *args.Fn
	Method     string
	Verb       string
	URL        url.URL
	Name       string
	handler    http.HandlerFunc // If Fn is defined, Target is ignored (for generated code)
	Generators *map[string][]args.Gen
	Layout     *args.Fn

	Controller     any
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	ParamName      string
}

func (a *Action) String() string {
	base := fmt.Sprintf("%9s %s %s", a.Verb, a.URL.String(), a.Name)
	opts := []string{}

	if a.ControllerName != "" {
		opts = append(opts, a.ControllerName)
	}
	if a.Plural != "" {
		opts = append(opts, a.Plural)
	}
	if a.Singular != "" {
		opts = append(opts, a.Singular)
	}
	if a.ParamName != "" {
		opts = append(opts, a.ParamName)
	}

	return base + " " + strings.Join(opts, " ")
}

func (a *Action) values(w http.ResponseWriter, r *http.Request) (args.InputSet, *Context) {
	ctx := newContext(w, r)
	return args.InputSet{
		Values: map[string][]any{
			"http.ResponseWriter": {w},
			"*http.Request":       {r},
			"string":              extractParam(r.URL.Path, a.URL.Path),
			"*lazyassets.Assets":  {a.Assets},
			"*lazyaction.Context": {ctx},
			"lazyaction.Context":  {*ctx},
			"*url.URL":            {r.URL},
			"url.URL":             {*r.URL},
			"[]lazyaction.Route":  {a.Dispatcher.Routes()},
		},
		Generators: *a.Generators,
	}, ctx
}

func (a *Action) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var body []byte
	var done bool
	var header http.Header

	if a.handler != nil {
		a.handler(w, r)
		return
	}
	rec := actionrecorder.New(w)
	ins, _ := a.values(rec, r)

	// If we have a function, call it. Here we dont have filters neighter generators
	if a.fn != nil {
		outs, err := a.fn.Call(ins)
		if err != nil {
			panic(err)
		}
		a.processOutputs(rec, r, a.fn.Outs, outs)
		rec.Send()
		return
	}

	//
	cType := reflect.TypeOf(a.Controller).Elem()
	instance := reflect.New(cType).Interface()

	instanceV := reflect.ValueOf(instance)
	method := instanceV.Method(a.methodI)

	fn := args.NewFn(method)

	if c, ok := (instance).(setVars); ok {
		c.setVars(rec, r, a, a.Assets)
	}

	// Call the method
	outs, err := fn.Call(ins)
	// If some generator returns an error, we will have it here
	// The generator error might be able to handle itself.
	if h, ok := err.(http.Handler); ok {
		h.ServeHTTP(rec, r)
		rec.Send()
		return
	}
	if err != nil {
		rec.WriteHeader(500)
		body = []byte(err.Error())
		if a.Layout != nil {
			goto Layout
		}

		rec.Write(body)
		rec.Send()
		return
	}
	done, _ = a.processOutputs(rec, r, fn.Outs, outs)

	if done || a.Layout == nil {
		rec.Send()
		return
	}
	header = w.Header()
	header.Del("Content-Length")
	body = rec.Bytes()

Layout:
	rec.B.Reset()

	ctx := newContext(rec, r)

	ins.Merge(args.InputSet{
		Values: map[string][]any{
			"http.ResponseWriter": {rec},
			"*lazyaction.Context": {ctx},
			"[]uint8":             {body},
		},
	})
	method = instanceV.MethodByName("RenderLayout")
	fn = args.NewFn(method)
	outs, err = fn.Call(ins)
	if err != nil {
		panic(fmt.Errorf("error calling layout: %w (%q)", err, a.Name))
	}
	if rec.S != 0 {
		w.WriteHeader(rec.S)
	}
	a.processOutputs(w, r, a.Layout.Outs, outs)
}

func (a *Action) processOutputs(w http.ResponseWriter, r *http.Request, outsT []string, outs []reflect.Value) (done bool, err error) {
	for i, outType := range outsT {
		out := outs[i]
		switch outType {
		case "[]byte", "[]uint8":
			w.Write(out.Bytes())
		case "io.WriterTo":
			writer := out.Interface()
			if writer != nil {
				writer.(io.WriterTo).WriteTo(w)
			}
		case "string":
			s := out.String()
			w.Write([]byte(s))
		case "int":
			w.WriteHeader(int(out.Int()))
		case "lazyaction.Result", "http.Handler":
			if out.IsNil() {
				continue
			}
			out.Interface().(http.Handler).ServeHTTP(w, r)
			return true, nil

		case "error":
			if out.IsNil() {
				continue
			}
			err = out.Interface().(error)
			if err != nil {
				if redirect, ok := err.(http.Handler); ok {
					redirect.ServeHTTP(w, r)
					return true, nil
				}
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
		default:
			panic("Unknown return type: " + outType)
		}
	}
	return false, nil
}

func extractParam(url, path string) []any {

	out := []any{}
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
