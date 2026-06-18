package actioncall

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Options struct {
	RoutePath string
}

type Plan struct {
	action     reflect.Value
	args       []resolver
	generators map[reflect.Type]*generatorPlan
	paramNames []string
	standard   bool
}

type generatorPlan struct {
	method   reflect.Value
	name     string
	args     []resolver
	compiled bool
}

type resolver interface {
	resolve(*requestState) (reflect.Value, error)
}

type requestState struct {
	plan       *Plan
	controller reflect.Value
	w          http.ResponseWriter
	r          *http.Request
	cache      map[reflect.Type]reflect.Value
	resolving  map[reflect.Type]bool
}

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	requestType = reflect.TypeOf((*http.Request)(nil))
	writerType  = reflect.TypeOf((*http.ResponseWriter)(nil)).Elem()
)

func Compile(controllerType reflect.Type, action reflect.Value, opts Options) (*Plan, error) {
	if !action.IsValid() {
		return nil, fmt.Errorf("action is nil")
	}
	actionType := action.Type()
	if err := validateActionType(controllerType, actionType); err != nil {
		return nil, err
	}

	plan := &Plan{
		action:     action,
		paramNames: routeParamNames(opts.RoutePath),
	}
	if standardAction(controllerType, actionType) {
		plan.standard = true
		return plan, nil
	}

	var err error
	plan.generators, err = collectGenerators(controllerType)
	if err != nil {
		return nil, err
	}

	compiler := compileState{plan: plan}
	args, err := compiler.compileInputs(actionType, 1, "action", nil)
	if err != nil {
		return nil, err
	}
	plan.args = args
	return plan, nil
}

func (p *Plan) Call(controller reflect.Value, w http.ResponseWriter, r *http.Request) error {
	if p.standard {
		return callErrorOutput(p.action.Call([]reflect.Value{
			controller,
			reflect.ValueOf(w),
			reflect.ValueOf(r),
		}))
	}

	state := &requestState{
		plan:       p,
		controller: controller,
		w:          w,
		r:          r,
		cache:      map[reflect.Type]reflect.Value{},
		resolving:  map[reflect.Type]bool{},
	}
	inputs := make([]reflect.Value, 1, len(p.args)+1)
	inputs[0] = controller
	for _, resolver := range p.args {
		value, err := resolver.resolve(state)
		if err != nil {
			return err
		}
		inputs = append(inputs, value)
	}
	return callErrorOutput(p.action.Call(inputs))
}

func validateActionType(controllerType reflect.Type, actionType reflect.Type) error {
	if actionType.Kind() != reflect.Func ||
		actionType.NumIn() == 0 ||
		!controllerType.AssignableTo(actionType.In(0)) ||
		actionType.NumOut() != 1 ||
		!actionType.Out(0).Implements(errorType) {
		return fmt.Errorf("controller action must have signature func(*Controller, ...) error")
	}
	return nil
}

func standardAction(controllerType reflect.Type, actionType reflect.Type) bool {
	return actionType.NumIn() == 3 &&
		controllerType.AssignableTo(actionType.In(0)) &&
		actionType.In(1) == writerType &&
		actionType.In(2) == requestType
}

func collectGenerators(controllerType reflect.Type) (map[reflect.Type]*generatorPlan, error) {
	generators := map[reflect.Type]*generatorPlan{}
	for i := 0; i < controllerType.NumMethod(); i++ {
		method := controllerType.Method(i)
		if !generatorName(method.Name) {
			continue
		}
		methodType := method.Type
		if methodType.NumOut() == 0 || methodType.NumOut() > 2 {
			return nil, fmt.Errorf("%s must return T or (T, error)", method.Name)
		}
		if methodType.NumOut() == 2 && !methodType.Out(1).Implements(errorType) {
			return nil, fmt.Errorf("%s second return value must be error", method.Name)
		}
		out := methodType.Out(0)
		if _, exists := generators[out]; exists {
			return nil, fmt.Errorf("multiple generators return %s", out)
		}
		generators[out] = &generatorPlan{
			method: method.Func,
			name:   method.Name,
		}
	}
	return generators, nil
}

func generatorName(name string) bool {
	if !strings.HasPrefix(name, "Gen") || len(name) == len("Gen") {
		return false
	}
	next := name[len("Gen"):]
	r, _ := utf8.DecodeRuneInString(next)
	return r != utf8.RuneError && unicode.IsUpper(r)
}

type compileState struct {
	plan      *Plan
	compiling map[reflect.Type]bool
}

func (c *compileState) compileInputs(fn reflect.Type, start int, label string, stack []reflect.Type) ([]resolver, error) {
	paramCursor := 0
	args := make([]resolver, 0, fn.NumIn()-start)
	for i := start; i < fn.NumIn(); i++ {
		resolver, err := c.compileResolver(fn.In(i), &paramCursor, label, stack)
		if err != nil {
			return nil, err
		}
		args = append(args, resolver)
	}
	return args, nil
}

func (c *compileState) compileResolver(t reflect.Type, paramCursor *int, label string, stack []reflect.Type) (resolver, error) {
	switch t {
	case writerType:
		return writerResolver{}, nil
	case requestType:
		return requestResolver{}, nil
	case contextType:
		return contextResolver{}, nil
	}

	if generator, ok := c.plan.generators[t]; ok {
		if err := c.compileGenerator(t, generator, stack); err != nil {
			return nil, err
		}
		return generatorResolver{t: t}, nil
	}

	if routeConvertible(t) {
		if *paramCursor >= len(c.plan.paramNames) {
			return nil, fmt.Errorf("%s needs %s but route has only %d named parameter(s)", label, t, len(c.plan.paramNames))
		}
		name := c.plan.paramNames[*paramCursor]
		*paramCursor = *paramCursor + 1
		return routeParamResolver{name: name, t: t}, nil
	}

	return nil, fmt.Errorf("%s parameter %s has no resolver", label, t)
}

func (c *compileState) compileGenerator(t reflect.Type, generator *generatorPlan, stack []reflect.Type) error {
	if generator.compiled {
		return nil
	}
	if c.compiling == nil {
		c.compiling = map[reflect.Type]bool{}
	}
	if c.compiling[t] {
		return fmt.Errorf("generator cycle: %s", formatTypeChain(append(stack, t)))
	}
	c.compiling[t] = true
	args, err := c.compileInputs(generator.method.Type(), 1, "generator "+generator.name, append(stack, t))
	delete(c.compiling, t)
	if err != nil {
		return err
	}
	generator.args = args
	generator.compiled = true
	return nil
}

type writerResolver struct{}

func (writerResolver) resolve(state *requestState) (reflect.Value, error) {
	return reflect.ValueOf(state.w), nil
}

type requestResolver struct{}

func (requestResolver) resolve(state *requestState) (reflect.Value, error) {
	return reflect.ValueOf(state.r), nil
}

type contextResolver struct{}

func (contextResolver) resolve(state *requestState) (reflect.Value, error) {
	return reflect.ValueOf(state.r.Context()), nil
}

type generatorResolver struct {
	t reflect.Type
}

func (r generatorResolver) resolve(state *requestState) (reflect.Value, error) {
	if value, ok := state.cache[r.t]; ok {
		return value, nil
	}
	if state.resolving[r.t] {
		return reflect.Value{}, fmt.Errorf("generator cycle resolving %s", r.t)
	}
	generator := state.plan.generators[r.t]
	if generator == nil {
		return reflect.Value{}, fmt.Errorf("generator for %s was not compiled", r.t)
	}

	state.resolving[r.t] = true
	inputs := make([]reflect.Value, 1, len(generator.args)+1)
	inputs[0] = state.controller
	for _, resolver := range generator.args {
		value, err := resolver.resolve(state)
		if err != nil {
			delete(state.resolving, r.t)
			return reflect.Value{}, err
		}
		inputs = append(inputs, value)
	}
	outputs := generator.method.Call(inputs)
	delete(state.resolving, r.t)

	if len(outputs) == 2 {
		if err := valueAsError(outputs[1]); err != nil {
			return reflect.Value{}, err
		}
	}
	value := outputs[0]
	state.cache[r.t] = value
	return value, nil
}

type routeParamResolver struct {
	name string
	t    reflect.Type
}

func (r routeParamResolver) resolve(state *requestState) (reflect.Value, error) {
	raw := state.r.PathValue(r.name)
	return convertRouteParam(raw, r.name, r.t)
}

func routeConvertible(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func convertRouteParam(raw string, name string, t reflect.Type) (reflect.Value, error) {
	value := reflect.New(t).Elem()
	switch t.Kind() {
	case reflect.String:
		value.SetString(raw)
		return value, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, t.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("route parameter %q value %q cannot convert to %s: %w", name, raw, t, err)
		}
		value.SetInt(parsed)
		return value, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(raw, 10, t.Bits())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("route parameter %q value %q cannot convert to %s: %w", name, raw, t, err)
		}
		value.SetUint(parsed)
		return value, nil
	default:
		return reflect.Value{}, fmt.Errorf("route parameter %q cannot convert to %s", name, t)
	}
}

func callErrorOutput(outputs []reflect.Value) error {
	if len(outputs) != 1 {
		return fmt.Errorf("action returned %d values, want one error", len(outputs))
	}
	return valueAsError(outputs[0])
}

func valueAsError(value reflect.Value) error {
	if !value.IsValid() {
		return nil
	}
	if isNilable(value.Kind()) && value.IsNil() {
		return nil
	}
	if err, ok := value.Interface().(error); ok {
		return err
	}
	return nil
}

func isNilable(kind reflect.Kind) bool {
	switch kind {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

func routeParamNames(path string) []string {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	names := make([]string, 0, len(segments))
	for _, segment := range segments {
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(segment, "{"), "}")
		if name == "$" {
			continue
		}
		name = strings.TrimSuffix(name, "...")
		if strings.TrimSpace(name) == "" {
			continue
		}
		names = append(names, name)
	}
	return names
}

func formatTypeChain(types []reflect.Type) string {
	parts := make([]string, 0, len(types))
	for _, t := range types {
		parts = append(parts, t.String())
	}
	return strings.Join(parts, " -> ")
}
