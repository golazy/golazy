package lazyroutes

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyview"
)

type Action func(http.ResponseWriter, *http.Request) error

type controllerConstructor struct {
	value          reflect.Value
	controllerType reflect.Type
}

func newControllerConstructor(controller any) controllerConstructor {
	constructorValue := reflect.ValueOf(controller)
	if !constructorValue.IsValid() {
		panic("lazyroutes: controller is nil")
	}
	if constructorValue.Kind() == reflect.Func && constructorValue.IsNil() {
		panic("lazyroutes: controller is nil")
	}

	constructorType := constructorValue.Type()
	contextType := reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType := reflect.TypeOf((*error)(nil)).Elem()

	if constructorType.Kind() != reflect.Func ||
		constructorType.NumIn() != 1 ||
		!constructorType.In(0).Implements(contextType) ||
		constructorType.NumOut() != 2 ||
		constructorType.Out(0).Kind() != reflect.Pointer ||
		!constructorType.Out(1).Implements(errorType) {
		panic(fmt.Errorf("lazyroutes: controller must have signature func(context.Context) (*Controller, error)"))
	}

	return controllerConstructor{
		value:          constructorValue,
		controllerType: constructorType.Out(0),
	}
}

func (c controllerConstructor) bind(ctx context.Context, action reflect.Value) http.Handler {
	validateControllerAction(c.controllerType, action)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = ensureResponseState(w)
		var controller any
		defer func() {
			if recovered := recover(); recovered != nil {
				reportControllerError(ctx, w, r, controller, lazycontroller.PanicError(recovered))
			}
		}()

		controllerContext := lazycontroller.WithWriter(ctx, w)
		controllerContext = lazycontroller.WithRequest(controllerContext, r)
		if route, params, ok := RouteFromRequest(r); ok {
			controllerContext = lazycontroller.WithRoute(controllerContext, lazyview.Route{
				Name:       route.Name,
				Method:     route.Method,
				Path:       route.Path,
				Namespace:  route.Namespace,
				Controller: route.Controller,
				Action:     route.Action,
				Params:     params,
			})
		}

		values := c.value.Call([]reflect.Value{reflect.ValueOf(controllerContext)})
		if !values[1].IsNil() {
			reportControllerError(ctx, w, r, nil, values[1].Interface().(error))
			return
		}

		controller = values[0].Interface()
		lazycontroller.ReportController(r, controller)
		values = action.Call([]reflect.Value{
			values[0],
			reflect.ValueOf(w),
			reflect.ValueOf(r),
		})
		if !values[0].IsNil() {
			reportControllerError(ctx, w, r, controller, values[0].Interface().(error))
			return
		}

		if lazycontroller.WasResponseSent(w) {
			return
		}
		if renderer, ok := controller.(interface{ Render(string) error }); ok {
			if err := renderer.Render(""); err != nil {
				reportControllerError(ctx, w, r, controller, err)
			}
		}
	})
}

func actionValue(action any) reflect.Value {
	actionValue := reflect.ValueOf(action)
	if !actionValue.IsValid() {
		panic("lazyroutes: controller action is nil")
	}
	if actionValue.Kind() == reflect.Func && actionValue.IsNil() {
		panic("lazyroutes: controller action is nil")
	}
	return actionValue
}

func validateControllerAction(controllerType reflect.Type, actionValue reflect.Value) {
	actionType := actionValue.Type()
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	writerType := reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
	requestType := reflect.TypeOf((*http.Request)(nil))

	if actionType.Kind() != reflect.Func ||
		actionType.NumIn() != 3 ||
		!controllerType.AssignableTo(actionType.In(0)) ||
		!actionType.In(1).Implements(writerType) ||
		actionType.In(2) != requestType ||
		actionType.NumOut() != 1 ||
		!actionType.Out(0).Implements(errorType) {
		panic(fmt.Errorf("lazyroutes: controller action must have signature func(*Controller, http.ResponseWriter, *http.Request) error"))
	}
}

func Handle(action Action) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				err := lazycontroller.PanicError(recovered)
				if lazycontroller.ReportError(r, nil, err) {
					return
				}
				lazycontroller.WriteError(w, r, err)
			}
		}()
		if err := action(w, r); err != nil {
			if lazycontroller.ReportError(r, nil, err) {
				return
			}
			lazycontroller.WriteError(w, r, err)
		}
	})
}

type controllerErrorHandler interface {
	HandleError(http.ResponseWriter, *http.Request, error) error
}

type responseTracker struct {
	http.ResponseWriter
	sent bool
}

func ensureResponseState(w http.ResponseWriter) http.ResponseWriter {
	if lazycontroller.WasResponseSent(w) {
		return w
	}
	if _, ok := w.(interface{ WasResponseSent() bool }); ok {
		return w
	}
	return &responseTracker{ResponseWriter: w}
}

func (w *responseTracker) Write(data []byte) (int, error) {
	w.sent = true
	return w.ResponseWriter.Write(data)
}

func (w *responseTracker) WriteHeader(status int) {
	if w.sent {
		return
	}
	w.sent = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseTracker) WasResponseSent() bool {
	return w.sent || lazycontroller.WasResponseSent(w.ResponseWriter)
}

func (w *responseTracker) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func reportControllerError(ctx context.Context, w http.ResponseWriter, r *http.Request, controller any, err error) {
	if lazycontroller.ReportError(r, controller, err) {
		return
	}
	handleControllerError(ctx, w, r, controller, err)
}

func handleControllerError(ctx context.Context, w http.ResponseWriter, r *http.Request, controller any, err error) {
	if err == nil {
		return
	}
	lazycontroller.ResetResponse(w)
	if handler, ok := controller.(controllerErrorHandler); ok {
		handleErr := callControllerErrorHandler(handler, w, r, err)
		if handleErr == nil {
			return
		}
		lazycontroller.ResetResponse(w)
		if lazycontroller.WriteErrorFallback(ctx, w, r) {
			return
		}
		lazycontroller.WriteError(w, r, handleErr)
		return
	}
	lazycontroller.WriteError(w, r, err)
}

func callControllerErrorHandler(handler controllerErrorHandler, w http.ResponseWriter, r *http.Request, err error) (handleErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			handleErr = lazycontroller.PanicError(recovered)
		}
	}()
	return handler.HandleError(w, r, err)
}
