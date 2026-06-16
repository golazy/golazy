package lazyview

import (
	"fmt"
	"reflect"
)

// Helper is the native context-aware helper function shape.
type Helper func(*Context, ...any) (any, error)

// AddHelpers adds helpers to the view set.
func (v *Views) AddHelpers(helpers map[string]any) {
	if len(helpers) == 0 {
		return
	}
	if v.Helpers == nil {
		v.Helpers = map[string]any{}
	}
	for name, helper := range helpers {
		if name == "" {
			panic("lazyview: helper name is required")
		}
		if helper == nil {
			panic(fmt.Sprintf("lazyview: helper %q is nil", name))
		}
		v.Helpers[name] = helper
	}
	v.clearCache()
}

// Helper adds one helper to the view set.
func (v *Views) Helper(name string, helper any) {
	v.AddHelpers(map[string]any{name: helper})
}

func (v *Views) baseHelpers() map[string]any {
	return map[string]any{
		"partial": Helper(func(ctx *Context, args ...any) (any, error) {
			return ctx.partial(args...)
		}),
	}
}

func bindHelper(ctx *Context, helper any) any {
	switch helper := helper.(type) {
	case Helper:
		return func(args ...any) (any, error) {
			return helper(ctx, args...)
		}
	case func(*Context, ...any) (any, error):
		return func(args ...any) (any, error) {
			return helper(ctx, args...)
		}
	case func(*Context, ...any) any:
		return func(args ...any) any {
			return helper(ctx, args...)
		}
	}

	value := reflect.ValueOf(helper)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return helper
	}

	contextType := reflect.TypeOf((*Context)(nil))
	helperType := value.Type()
	if helperType.NumIn() == 0 || helperType.In(0) != contextType {
		return helper
	}

	return func(args ...any) (any, error) {
		values := make([]reflect.Value, 0, len(args)+1)
		values = append(values, reflect.ValueOf(ctx))

		fixedInputs := helperType.NumIn()
		variadic := helperType.IsVariadic()
		if !variadic && len(args) != fixedInputs-1 {
			return nil, fmt.Errorf("lazyview: helper expects %d arguments, got %d", fixedInputs-1, len(args))
		}
		if variadic && len(args) < fixedInputs-2 {
			return nil, fmt.Errorf("lazyview: helper expects at least %d arguments, got %d", fixedInputs-2, len(args))
		}

		for index, arg := range args {
			inputIndex := index + 1
			var inputType reflect.Type
			if variadic && inputIndex >= fixedInputs-1 {
				inputType = helperType.In(fixedInputs - 1).Elem()
			} else {
				inputType = helperType.In(inputIndex)
			}
			value, err := helperArgument(inputType, arg)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}

		results := value.Call(values)
		return helperResults(results)
	}
}

func helperArgument(inputType reflect.Type, arg any) (reflect.Value, error) {
	if arg == nil {
		return reflect.Zero(inputType), nil
	}
	value := reflect.ValueOf(arg)
	if value.Type().AssignableTo(inputType) {
		return value, nil
	}
	if value.Type().ConvertibleTo(inputType) {
		return value.Convert(inputType), nil
	}
	return reflect.Value{}, fmt.Errorf("lazyview: helper argument %s is not assignable to %s", value.Type(), inputType)
}

func helperResults(results []reflect.Value) (any, error) {
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		return results[0].Interface(), nil
	case 2:
		if !results[1].IsNil() {
			err, ok := results[1].Interface().(error)
			if !ok {
				return nil, fmt.Errorf("lazyview: helper second return value must be an error")
			}
			return nil, err
		}
		return results[0].Interface(), nil
	default:
		return nil, fmt.Errorf("lazyview: helper returned too many values")
	}
}

func copyHelpers(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(source))
	for name, helper := range source {
		out[name] = helper
	}
	return out
}
