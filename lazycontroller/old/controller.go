package lazycontroller

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"

	"github.com/gorilla/sessions"
	"golazy.dev/lazyassets"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazysupport"
	"golazy.dev/lazyview"
)

type Controller struct {
	done              bool
	params            Params
	written           bool
	ctx               context.Context
	Request           *http.Request
	ResponseWriter    http.ResponseWriter
	assts             *lazyassets.Storage
	route             *lazydispatch.Route
	dispatcher        *lazydispatch.Dispatcher
	views             *lazyview.Views
	session           *sessions.Session
	sessionHasChanges bool
	ControllerName    string
	ActionName        string
	Namespace         string
	csrf              string

	ViewVariables []any

	Layout string
	W      http.ResponseWriter
}

type TemplateVar struct {
	k string
	v any
}

func (c *Controller) renderVars(data ...any) map[string]any {
	vars := toMap(append(c.ViewVariables, data...)...)
	vars["Controller"] = c.ControllerName
	vars["Action"] = c.ActionName
	vars["Namespace"] = c.Namespace
	//vars["Helpers"] = Helpers(*c)
	//vars["Time"] = &Time{time.Now()}
	return vars
}

func objToMap(data any) map[string]any {
	vars := map[string]any{}
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic(fmt.Sprintf("data must be a struct, not %s", t.Kind()))
	}

	for _, field := range reflect.VisibleFields(t) {
		if field.IsExported() {
			vars[field.Name] = v.Field(field.Index[0]).Interface()
		}
	}
	return vars
}

func toMap(data ...any) map[string]any {
	vars := map[string]any{}

	for _, d := range data {

		switch value := d.(type) {
		case TemplateObj:

			for k, v := range objToMap(value.obj) {
				vars[k] = v
			}
			continue
		case map[string]any:
			for k, v := range value {
				vars[k] = v
			}
			continue
		case TemplateVar:
			vars[value.k] = value.v
			continue
		}

		t := reflect.TypeOf(d)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			vars[t.Name()] = d
		case reflect.Slice:
			sliceType := t.Elem()
			for sliceType.Kind() == reflect.Ptr {
				sliceType = sliceType.Elem()
			}
			vars[lazysupport.Pluralize(sliceType.Name())] = d
		default:
			panic(fmt.Sprintf("when rendering views, found data that is not a struct or slice: %T", d))
		}

	}
	return vars

}

func Var(key string, value any) TemplateVar {
	return TemplateVar{k: key, v: value}
}

type TemplateObj struct {
	obj any
}

func Obj(value any) TemplateObj {
	return TemplateObj{value}
}

type writeDetector struct {
	http.ResponseWriter
	written *bool
}

func (w *writeDetector) WriteHeader(code int) {
	*w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *writeDetector) Write(b []byte) (int, error) {
	*w.written = true
	return w.ResponseWriter.Write(b)
}
func (w *writeDetector) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	*w.written = true
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (c *Controller) Render() {
	c.views.Render(lazyview.Options{
		Ctx:        c.ctx,
		Writer:     c.ResponseWriter,
		Variables:  c.renderVars(),
		UseLayout:  true,
		Controller: c.ControllerName,
		Action:     c.ActionName,
		Namespace:  c.Namespace,
	})
}

func (c *Controller) RenderContent(content string) {
	c.views.Render(lazyview.Options{
		Content:    content,
		Ctx:        c.ctx,
		Writer:     c.ResponseWriter,
		Variables:  c.renderVars(content),
		UseLayout:  false,
		Controller: c.ControllerName,
		Action:     c.ActionName,
		Namespace:  c.Namespace,
	})

}

func (c *Controller) After_ZZZZ_EnsureRender() {
	if !c.written {
		c.Render()
	}
}

func (c *Controller) Before_0000_SetupRequest(r *http.Request, w http.ResponseWriter, a *lazyassets.Storage, route *lazydispatch.Route, dispatcher *lazydispatch.Dispatcher, views *lazyview.Views) {
	c.Request = r
	c.ResponseWriter = &writeDetector{ResponseWriter: w, written: &c.written}
	c.assts = a
	c.ctx = r.Context()
	c.route = route
	c.views = views

	c.dispatcher = dispatcher
	c.ControllerName = route.Controller
	c.ActionName = route.Action
	c.Namespace = route.Namespace

	c.initSession()
	c.ensureCSRFToken()
}

func (v *Controller) ViewVar(name string, val any) {
	v.ViewVariables = append(v.ViewVariables, Var(name, val))
}

func (v *Controller) ViewObj(val any) {
	v.ViewVariables = append(v.ViewVariables, Obj(val))
}
func (v *Controller) ViewData(val any) {
	v.ViewVariables = append(v.ViewVariables, val)
}

var (
	ErrBodyAlreadySent error = fmt.Errorf("body already sent")
)

func (c *Controller) RedirectBack(fallback string, code ...int) {
	to := c.Request.Referer()
	if to == "" {
		to = fallback
	}
	status := http.StatusSeeOther
	if len(code) > 0 {
		status = code[0]
	}
	http.Redirect(c.ResponseWriter, c.Request, to, status)

}
func (c *Controller) Redirect(url string, code ...int) {
	c.assertBodyNotSent()

	status := http.StatusSeeOther
	if len(code) > 0 {
		status = code[0]
	}

	http.Redirect(c.ResponseWriter, c.Request, url, status)
	c.done = true
}

func (c *Controller) RedirectTo(args ...any) {
	c.Redirect(c.PathTo(args...))
}

func (c *Controller) PathTo(args ...any) string {
	return c.dispatcher.PathFor(args...)
}

func (c *Controller) assertBodyNotSent() {
	if c.written {
		panic(ErrBodyAlreadySent)
	}
}

func (c *Controller) NoContent() {
	c.assertBodyNotSent()
	c.ResponseWriter.WriteHeader(http.StatusNoContent)
}

func (c *Controller) UnprocessableEntity() {
	c.assertBodyNotSent()
	c.ResponseWriter.WriteHeader(http.StatusUnprocessableEntity)
}

func (c *Controller) Forbidden() {
	c.assertBodyNotSent()
	c.ResponseWriter.WriteHeader(http.StatusForbidden)
}

func (c *Controller) BadRequest() {
	c.assertBodyNotSent()
	c.ResponseWriter.WriteHeader(http.StatusBadRequest)
}

func (c *Controller) SendFile(filename string, data io.Reader) {
	c.ResponseWriter.Header().Set("Content-Disposition", "attachment; filename=\""+filename)
	io.Copy(c.ResponseWriter, data)
}
