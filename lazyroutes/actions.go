package lazyroutes

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"golazy.dev/lazycontroller"
	"golazy.dev/lazyroutes/actioncall"
	"golazy.dev/lazytelemetry"
	"golazy.dev/lazytelemetry/lazytracing"
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
		constructorType.Out(0).Elem().Kind() != reflect.Struct ||
		!constructorType.Out(1).Implements(errorType) {
		panic(fmt.Errorf("lazyroutes: controller must have signature func(context.Context) (*Controller, error)"))
	}

	return controllerConstructor{
		value:          constructorValue,
		controllerType: constructorType.Out(0),
	}
}

func (c controllerConstructor) bind(ctx context.Context, routePath string, action reflect.Value) http.Handler {
	actionPlan, err := actioncall.Compile(c.controllerType, action, actioncall.Options{RoutePath: routePath})
	if err != nil {
		panic(fmt.Errorf("lazyroutes: bind controller action: %w", err))
	}
	beforePlan, hasBeforeAction, err := actioncall.CompileMethod(c.controllerType, "BeforeAction", actioncall.Options{RoutePath: routePath})
	if err != nil {
		panic(fmt.Errorf("lazyroutes: bind controller before action: %w", err))
	}
	prototype, err := c.construct(ctx)
	if err != nil {
		panic(fmt.Errorf("lazyroutes: initialize controller: %w", err))
	}

	binding := &controllerBinding{
		ctx:             ctx,
		actionPlan:      actionPlan,
		beforePlan:      beforePlan,
		hasBeforeAction: hasBeforeAction,
		prototype:       prototype,
	}
	binding.pool = sync.Pool{
		New: func() any {
			instance := reflect.New(c.controllerType.Elem())
			instance.Elem().Set(prototype.Elem())
			return instance
		},
	}
	return binding
}

func (c controllerConstructor) construct(ctx context.Context) (reflect.Value, error) {
	values := c.value.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if !values[1].IsNil() {
		return reflect.Value{}, values[1].Interface().(error)
	}
	if values[0].IsNil() {
		return reflect.Value{}, fmt.Errorf("lazyroutes: controller constructor returned nil")
	}
	return values[0], nil
}

type controllerBinding struct {
	ctx             context.Context
	actionPlan      *actioncall.Plan
	beforePlan      *actioncall.Plan
	hasBeforeAction bool
	prototype       reflect.Value
	pool            sync.Pool
}

func (b *controllerBinding) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = ensureResponseState(w)
	route := routeForRequest(r)
	ctx, dispatchSpan := startRouteRegion(r.Context(), "dispatch", route)
	if dispatchSpan != nil {
		defer dispatchSpan.End()
		r = r.WithContext(ctx)
	}
	var controller any
	var controllerValue reflect.Value
	defer func() {
		if recovered := recover(); recovered != nil {
			handleControllerError(b.ctx, w, r, controller, lazycontroller.PanicError(recovered))
		}
		if controller != nil {
			if resetter, ok := controller.(lazycontroller.RequestResetter); ok {
				resetter.ResetRequest()
			}
		}
		if controllerValue.IsValid() {
			controllerValue.Elem().Set(b.prototype.Elem())
			b.pool.Put(controllerValue)
		}
	}()

	controllerReady := func() bool {
		_, controllerSpan := startRouteRegion(r.Context(), "controller", route)
		if controllerSpan != nil {
			defer controllerSpan.End()
		}

		controllerValue = b.pool.Get().(reflect.Value)
		controllerValue.Elem().Set(b.prototype.Elem())
		controller = controllerValue.Interface()
		if binder, ok := controller.(lazycontroller.RequestBinder); ok {
			if err := binder.BindRequest(w, r, route); err != nil {
				if controllerSpan != nil {
					controllerSpan.RecordError(err)
				}
				handleControllerError(b.ctx, w, r, controller, err)
				return false
			}
		}
		if requester, ok := controller.(interface{ Request() *http.Request }); ok {
			if request := requester.Request(); request != nil {
				r = request
			}
		}
		lazycontroller.ReportController(r, controller)
		if b.hasBeforeAction {
			if err := b.beforePlan.Call(controllerValue, w, r); err != nil {
				if controllerSpan != nil {
					controllerSpan.RecordError(err)
				}
				handleControllerError(b.ctx, w, r, controller, err)
				return false
			}
		}
		return true
	}()
	if !controllerReady {
		return
	}

	actionRequest, actionSpan := requestRouteRegion(r, "controller.action", route)
	actionErr := callWithActionRegion(actionSpan, func() error {
		return b.actionPlan.Call(controllerValue, w, actionRequest)
	})
	if actionErr != nil {
		handleControllerError(b.ctx, w, r, controller, actionErr)
		return
	}

	if lazycontroller.WasResponseSent(w) {
		return
	}
	if renderer, ok := controller.(interface{ RenderHTML(string) error }); ok {
		if err := renderer.RenderHTML(""); err != nil {
			handleControllerError(b.ctx, w, r, controller, err)
		}
	}
}

func routeForRequest(r *http.Request) lazyview.Route {
	route, params, ok := RouteFromRequest(r)
	if !ok {
		return lazyview.Route{}
	}
	return lazyview.Route{
		Name:       route.Name,
		Method:     route.Method,
		Path:       route.Path,
		Namespace:  route.Namespace,
		Controller: route.Controller,
		Action:     route.Action,
		Params:     params,
	}
}

func routeOperationName(kind string, route lazyview.Route) string {
	controller := strings.TrimSpace(route.Controller)
	action := strings.TrimSpace(route.Action)
	switch {
	case controller != "" && action != "":
		return kind + " " + controller + "." + action
	case controller != "":
		return kind + " " + controller
	case action != "":
		return kind + " " + action
	default:
		return kind
	}
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

func Handle(action Action) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r, dispatchSpan := requestRegion(r, "dispatch")
		if dispatchSpan != nil {
			defer dispatchSpan.End()
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				err := lazycontroller.PanicError(recovered)
				if dispatchSpan != nil {
					dispatchSpan.RecordError(err)
				}
				if lazycontroller.ReportError(r, nil, err) {
					return
				}
				lazycontroller.WriteError(w, r, err)
			}
		}()
		actionRequest, actionSpan := requestRegion(r, "action")
		actionErr := callWithActionRegion(actionSpan, func() error {
			return action(w, actionRequest)
		})
		if actionErr != nil {
			if lazycontroller.ReportError(r, nil, actionErr) {
				return
			}
			lazycontroller.WriteError(w, r, actionErr)
			return
		}
	})
}

func requestRegion(r *http.Request, name string, attrs ...slog.Attr) (*http.Request, *lazytracing.Span) {
	ctx, span := lazytelemetry.StartRegion(r.Context(), name, attrs...)
	if span == nil {
		return r, nil
	}
	return r.WithContext(ctx), span
}

func requestRouteRegion(r *http.Request, kind string, route lazyview.Route) (*http.Request, *lazytracing.Span) {
	ctx, span := startRouteRegion(r.Context(), kind, route)
	if span == nil {
		return r, nil
	}
	return r.WithContext(ctx), span
}

func startRouteRegion(ctx context.Context, kind string, route lazyview.Route) (context.Context, *lazytracing.Span) {
	if lazytelemetry.SpanFromContext(ctx) == nil {
		return ctx, nil
	}
	return lazytelemetry.StartRegion(ctx, routeOperationName(kind, route),
		slog.String("http.route", route.Path),
		slog.String("route.name", route.Name),
		slog.String("controller", route.Controller),
		slog.String("action", route.Action),
	)
}

func callWithActionRegion(span *lazytracing.Span, call func() error) (err error) {
	if span != nil {
		defer span.End()
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = lazycontroller.PanicError(recovered)
			if span != nil {
				span.RecordError(err)
			}
			panic(recovered)
		}
		if err != nil && span != nil {
			span.RecordError(err)
		}
	}()
	return call()
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
		if lazycontroller.DetailErrors(ctx) {
			lazycontroller.WriteErrorDetail(w, r, handleErr)
			return
		}
		if lazycontroller.WriteErrorFallback(ctx, w, r) {
			return
		}
		lazycontroller.WriteError(w, r, handleErr)
		return
	}
	if lazycontroller.DetailErrors(ctx) {
		lazycontroller.WriteErrorDetail(w, r, err)
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
