package lazyaction

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

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

func (a *Action) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if a.handler != nil {
		a.handler(w, r)
		return
	}
	rec := newActionRecorder(w)
	ctx := newContext(rec, r)
	ins := args.InputSet{
		Values: map[string][]any{
			"http.ResponseWriter": {rec},
			"*http.Request":       {r},
			"string":              extractParam(r.URL.Path, a.URL.Path),
			"*lazyassets.Assets":  {a.Assets},
			"*lazyaction.Context": {ctx},

			"[]lazyaction.Route": {a.Dispatcher.Routes()},
			"*url.URL":           {r.URL},
		},
		Generators: *a.Generators,
	}

	if a.fn != nil {

		outs, err := a.fn.Call(ins)
		if err != nil {
			panic(err)
		}
		a.processOutputs(rec, r, a.fn.Outs, outs)
		rec.Send()
		return
	}

	cType := reflect.TypeOf(a.Controller).Elem()
	instance := reflect.New(cType).Interface()

	instanceV := reflect.ValueOf(instance)
	method := instanceV.Method(a.methodI)

	fn := args.NewFn(method)

	if c, ok := (instance).(setVars); ok {
		c.setVars(w, r, a, a.Assets)
	}

	// Call the method
	outs, err := fn.Call(ins)
	if err != nil {
		panic(err)
	}
	a.processOutputs(rec, r, fn.Outs, outs)

	if a.Layout == nil {
		rec.Send()
		return
	}

	w.Header().Del("Content-Length")
	body := rec.Bytes()

	rec.B.Reset()

	ins = args.InputSet{
		Values: map[string][]any{
			"http.ResponseWriter": {rec},
			"*http.Request":       {r},
			"string":              extractParam(r.URL.Path, a.URL.Path),
			"*lazyassets.Assets":  {a.Assets},
			"*lazyaction.Context": {ctx},
			"[]lazyaction.Route":  {a.Dispatcher.Routes()},
			"[]uint8":             {body},
			"*url.URL":            {r.URL},
		},
		Generators: *a.Generators,
	}
	method = instanceV.MethodByName("RenderLayout")
	fn = args.NewFn(method)
	outs, err = fn.Call(ins)
	if err != nil {
		panic(fmt.Errorf("error calling layout: %w (%q)", err, a.Name))
	}
	a.processOutputs(w, r, a.Layout.Outs, outs)
	rec.Send()
}

func (a *Action) processOutputs(w http.ResponseWriter, r *http.Request, outsT []string, outs []reflect.Value) (err error) {
	for i, outType := range outsT {
		out := outs[i]
		switch outType {
		case "[]byte", "[]uint8":
			w.Write(out.Bytes())
		case "io.WriterTo":
			out.Interface().(io.WriterTo).WriteTo(w)
		case "string":
			s := out.String()
			w.Write([]byte(s))
		case "int":
			w.WriteHeader(int(out.Int()))
		case "error":
			if out.IsNil() {
				continue
			}
			err = out.Interface().(error)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
			}
		default:
			panic("Unknown return type: " + outType)
		}
	}
	return
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
