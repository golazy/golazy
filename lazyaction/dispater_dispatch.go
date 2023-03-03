package lazyaction

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
)

func (d *Dispatcher) dispatch(action *Action, w http.ResponseWriter, req *http.Request) {
	if action.Handler != nil {
		action.Handler(w, req)
		return
	}

	if action.Fn == nil {
		panic("no action defined")
	}

	ins := args.InputSet{
		Values: map[string][]any{
			"http.ResponseWriter":   {w},
			"*http.Request":         {req},
			"string":                extractParam(req.URL.Path, action.URL.Path),
			"*static_files.Manager": {d.Files},
		},
		Generators: *action.Generators,
	}

	if action.Layout == nil {
		outs, err := action.Fn.Call(ins)
		if err != nil {
			panic(err)
		}
		d.processOutputs(action.Fn, w, req, outs)
		return
	}

	// With layout, we record the output and pass it to the layout
	rec := newActionRecorder()
	ins.Values["http.ResponseWriter"] = []any{rec}
	outs, err := action.Fn.Call(ins)
	if err != nil {
		panic(err)
	}
	d.processOutputs(action.Fn, rec, req, outs)

	rec.Update(w)

	// Now call the layout
	ins.Values["http.ResponseWriter"] = []any{w}
	ins.Values["[]uint8"] = []any{rec.B.Bytes()}
	outs, err = action.Layout.Call(ins)
	if err != nil {
		panic(err)
	}
	d.processOutputs(action.Layout, w, req, outs)

}

type actionRecorder struct {
	H http.Header
	B *bytes.Buffer
	S int
}

func newActionRecorder() *actionRecorder {
	return &actionRecorder{
		H: http.Header{},
		B: &bytes.Buffer{},
	}
}

func (r *actionRecorder) Header() http.Header {
	return r.H
}
func (r *actionRecorder) Write(b []byte) (int, error) {
	return r.B.Write(b)
}
func (r *actionRecorder) WriteHeader(statusCode int) {
	r.S = statusCode
}

func (r *actionRecorder) Update(w http.ResponseWriter) {
	// Copy headers
	for k, v := range r.H {
		w.Header()[k] = v
	}
	// Copy status
	if r.S != 0 {
		w.WriteHeader(r.S)
	}

}

func (d *Dispatcher) processOutputs(fn *args.Fn, w http.ResponseWriter, r *http.Request, outs []reflect.Value) {
	for i, outType := range fn.Outs {
		out := outs[i]
		switch outType {
		case "[]byte":
			w.Write(out.Bytes())
		case "io.WriterTo":
			out.Interface().(io.WriterTo).WriteTo(w)
		case "string":
			s := out.String()
			w.Write([]byte(s))
		case "int":
			w.WriteHeader(int(out.Int()))
		case "error":
			err := out.Interface().(error)
			if err != nil {
				panic("error: " + err.Error())
			}
		default:
			panic("Unknown return type: " + outType)
		}
	}
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
