package lazyroutes

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

func (s *Scope) Get(path string, controller any, action any) {
	s.controllerAction(http.MethodGet, path, controller, action)
}

func (s *Scope) Post(path string, controller any, action any) {
	s.controllerAction(http.MethodPost, path, controller, action)
}

func (s *Scope) Put(path string, controller any, action any) {
	s.controllerAction(http.MethodPut, path, controller, action)
}

func (s *Scope) Patch(path string, controller any, action any) {
	s.controllerAction(http.MethodPatch, path, controller, action)
}

func (s *Scope) Delete(path string, controller any, action any) {
	s.controllerAction(http.MethodDelete, path, controller, action)
}

func (s *Scope) controllerAction(method string, path string, controller any, action any) {
	if s == nil {
		panic("lazyroutes: route scope is nil")
	}
	controllerConstructor := newControllerConstructor(controller)
	actionValue := actionValue(action)
	routePath := s.scopedPath(path)
	route := Route{
		Action:     actionName(action),
		Controller: controllerNameFromType(controllerConstructor.controllerType),
	}
	s.register(method, path, route, controllerConstructor.bind(s.Context, routePath, actionValue))
}

func actionName(v any) string {
	name := "action"
	if v == nil {
		return name
	}
	actionValue := reflect.ValueOf(v)
	if !actionValue.IsValid() || actionValue.Kind() != reflect.Func || actionValue.IsNil() {
		return name
	}
	fn := runtime.FuncForPC(actionValue.Pointer())
	if fn == nil {
		return name
	}
	name = fn.Name()
	if index := strings.LastIndex(name, "."); index >= 0 && index+1 < len(name) {
		name = name[index+1:]
	}
	return name
}
