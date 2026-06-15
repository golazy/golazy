package lazyroutes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"golazy.dev/lazycontroller"
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

	return Handle(func(w http.ResponseWriter, r *http.Request) error {
		values := c.value.Call([]reflect.Value{reflect.ValueOf(lazycontroller.WithWriter(ctx, w))})
		if !values[1].IsNil() {
			return values[1].Interface().(error)
		}

		values = action.Call([]reflect.Value{
			values[0],
			reflect.ValueOf(w),
			reflect.ValueOf(r),
		})
		if values[0].IsNil() {
			return nil
		}
		return values[0].Interface().(error)
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
		if err := action(w, r); err != nil {
			status := http.StatusInternalServerError
			var httpError *lazycontroller.HTTPError
			if errors.As(err, &httpError) {
				status = httpError.Status
			}
			http.Error(w, http.StatusText(status), status)
		}
	})
}
