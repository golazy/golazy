package lazyaction

import (
	"net/http"
	"reflect"
	"runtime"
	"strconv"
)

type action struct {
	Controller    *Controller
	TopController interface{}
	Verb          string
	Method        reflect.Value
	Path          string
	Function      string
	Destination   string
	Args          []string
	Returns       []string
}

func (a *action) String() string {
	return a.Controller.fullName + "#" + a.Function
}

func (a *action) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	inputs := make([]reflect.Value, len(a.Args))

	for i, argType := range a.Args {
		switch argType {
		case "string":
			inputs[i] = reflect.ValueOf("id")
		case "lazyaction.ResponseWriter":
			inputs[i] = reflect.ValueOf(ResponseWriter{w})
		case "lazyaction.Request":
			inputs[i] = reflect.ValueOf(Request{r})
		case "http.ResponseWriter":
			inputs[i] = reflect.ValueOf(w)
		case "*http.Request":
			inputs[i] = reflect.ValueOf(r)
		case "http.Request":
			panic("Should use *http.Request")
		default:
			f := runtime.FuncForPC(a.Method.Pointer())
			file, line := f.FileLine(f.Entry())

			panic(file + ":" + strconv.Itoa(line) + " Can't fill the argument of type " + argType + " for " + a.Controller.fullName + a.Destination)
		}
	}
	returns := a.Method.Call(inputs)
	if len(a.Returns) == 0 {
		return
	}
	var err error
	var body []byte

	for i, retType := range a.Returns {
		switch retType {
		case "error":
			err = reflect.ValueOf(returns[i]).Interface().(error)
		case "string":
			body = []byte(reflect.ValueOf(returns[i]).String())
		case "int":
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(body)
}
