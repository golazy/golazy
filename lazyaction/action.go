package lazyaction

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
)

func NewAction(method string, r *Resource) *Action {
	a := &Action{
		Resource: r,
		Method:   method,
	}

	a.Path, a.ParamsPosition = r.pathForMethod(method)
	a.Verb = r.verbForMethod(method)
	a.ResourceName, a.ActionName = r.nameForMethod(method)
	a.Destination = r.ControllerName + "#" + method

	a.RouteName = a.ResourceName
	if a.ActionName != "" {
		a.RouteName = a.ActionName + "_" + a.ResourceName
	}

	if a.Args != nil {
		panic("Initialize twice")
	}

	m := reflect.ValueOf(a.Controller).MethodByName(a.Method)
	if m.IsZero() {
		panic("Method " + a.Destination + " not found")
	}
	a.method = m

	mType := m.Type()

	for i := 0; i < mType.NumIn(); i++ {
		t := mType.In(i).String()
		if t == "string" {
			a.numInStrings++
		}
		a.Args = append(a.Args, t)
	}

	tCache := make(map[string]int)

	for i := 0; i < mType.NumOut(); i++ {
		t := mType.Out(i).String()
		tCache[t]++
		a.Rets = append(a.Rets, t)
	}
	if tCache["string"] > 1 {
		panic("method " + a.Destination + " returns more than one string")
	}
	if tCache["[]byte"] > 1 {
		panic("method " + a.Destination + " returns more than one []byte")
	}
	if tCache["[]byte"] > 0 && tCache["string"] > 0 {
		panic("method " + a.Destination + " can't return both []byte and string")
	}
	if tCache["error"] > 1 {
		panic("method " + a.Destination + " returns more than one error")
	}
	if tCache["int"] > 1 {
		panic("method " + a.Destination + " returns more than one int")
	}

	return a
}

type Action struct {
	*Resource
	Method         string // Member
	Verb           string
	Path           string
	RouteName      string
	ResourceName   string
	ActionName     string
	Destination    string
	ParamsPosition []int
	method         reflect.Value
	numInStrings   int
	Args           []string
	Rets           []string
}

func (a *Action) String() string {
	return fmt.Sprintf("%s %s %s %s", a.RouteName, a.Verb, a.Path, a.Destination)
}

func (a *Action) prepareArgs(ctx *Context) []reflect.Value {
	w := ctx.w
	r := ctx.r
	ins := make([]reflect.Value, len(a.Args))

	seenStrings := 0

	for i, t := range a.Args {
		switch t {
		case "string":
			arg := UrlExtractor(r.URL.Path).Extract(seenStrings, a.ParamsPosition)
			ins[i] = reflect.ValueOf(arg)
			seenStrings++
		case "lazyaction.ResponseWriter":
			ins[i] = reflect.ValueOf(ResponseWriter{w})
		case "*lazyaction.Request":
			ins[i] = reflect.ValueOf(&Request{r})
		case "lazyaction.Request":
			panic("Should use *http.Request")
		case "http.ResponseWriter":
			ins[i] = reflect.ValueOf(w)
		case "*http.Request":
			ins[i] = reflect.ValueOf(r)
		case "http.Request":
			panic("Should use *http.Request")
		case "*lazyaction.Session":
			ins[i] = reflect.ValueOf(&ctx.Session)
		case "*lazyaction.Context":
			ins[i] = reflect.ValueOf(ctx)
		default:
			panic(fmt.Sprintf("Can't fill the argument of type %s for %s", t, a.RouteName))
		}
	}

	return ins
}

func (a *Action) NewContext(w http.ResponseWriter, r *http.Request) (*Context, error) {
	c := &Context{
		Context: r.Context(),
		w:       w,
		r:       r,
	}
	err := c.loadFromRequest(r)
	return c, err
}

func (a *Action) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("Processing Request", r.URL.Path, a.Destination)

	ctx, err := a.NewContext(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	outs := a.method.Call(a.prepareArgs(ctx))

	for k := range ctx.headers {
		w.Header().Set(k, ctx.headers.Get(k))
	}

	content := []byte{}
	var wt io.WriterTo
	for i, t := range outs {
		switch a.Rets[i] {
		case "error":
			if t.IsNil() {
				continue
			}
			err := t.Interface().(error)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		case "string":
			content = []byte(t.String())
		case "[]uint8":
			content = t.Bytes()
		case "int":
			ctx.status = int(t.Int())
		case "io.WriterTo":
			if t.IsNil() {
				continue
			}
			wt = t.Interface().(io.WriterTo)
		default:
			panic(fmt.Sprintf("Can't fill the argument of type %s for %s#%s", a.Rets[i], a.RouteName, t))
		}
	}

	if ctx.Session.modified {
		err := ctx.s.Save(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if ctx.status != 0 {
		w.WriteHeader(ctx.status)
	}
	if wt != nil {
		wt.WriteTo(w)
	} else if len(content) > 0 {
		w.Write(content)
	}
}

type UrlExtractor string

func (u UrlExtractor) Extract(stringArg int, paramsPosition []int) string {

	components := strings.Split(string(u)[1:], "/")

	paramPos := len(paramsPosition) - 1 - stringArg
	if paramPos < 0 {
		return ""
	}
	pos := paramsPosition[paramPos]
	if pos >= len(components) {
		return ""
	}
	return components[pos]
}
