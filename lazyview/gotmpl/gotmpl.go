// Package gotmpl registers Go's html/template engine for lazyview.
package gotmpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"reflect"

	"golazy.dev/lazyview"
)

func init() {
	lazyview.RegisterEngine("tpl", func() lazyview.Engine {
		return &Engine{}
	})
}

// Engine renders .tpl files with html/template.
type Engine struct{}

// Render renders one template file.
func (e *Engine) Render(ctx *lazyview.Context, writer io.Writer, file string) error {
	data, err := fs.ReadFile(ctx.Views.FS, file)
	if err != nil {
		return err
	}
	tpl, err := template.New(file).
		Funcs(templateHelpers(ctx.HelperFuncs())).
		Parse(string(data))
	if err != nil {
		return err
	}
	return tpl.Execute(writer, templateVariables(ctx.Variables))
}

func templateHelpers(helpers map[string]any) template.FuncMap {
	funcs := template.FuncMap{}
	for name, helper := range helpers {
		value := reflect.ValueOf(helper)
		if !value.IsValid() || value.Kind() != reflect.Func {
			continue
		}
		funcs[name] = templateHelper(value)
	}
	return funcs
}

func templateHelper(helper reflect.Value) func(...any) (any, error) {
	helperType := helper.Type()
	return func(args ...any) (any, error) {
		fixedInputs := helperType.NumIn()
		variadic := helperType.IsVariadic()
		if !variadic && len(args) != fixedInputs {
			return nil, fmt.Errorf("gotmpl: helper expects %d arguments, got %d", fixedInputs, len(args))
		}
		if variadic && len(args) < fixedInputs-1 {
			return nil, fmt.Errorf("gotmpl: helper expects at least %d arguments, got %d", fixedInputs-1, len(args))
		}

		values := make([]reflect.Value, 0, len(args))
		for index, arg := range args {
			var inputType reflect.Type
			if variadic && index >= fixedInputs-1 {
				inputType = helperType.In(fixedInputs - 1).Elem()
			} else {
				inputType = helperType.In(index)
			}
			value, err := templateArgument(inputType, arg)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}

		result, err := templateResults(helper.Call(values))
		if err != nil {
			return nil, err
		}
		return templateValue(result), nil
	}
}

func templateArgument(inputType reflect.Type, arg any) (reflect.Value, error) {
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
	return reflect.Value{}, fmt.Errorf("gotmpl: helper argument %s is not assignable to %s", value.Type(), inputType)
}

func templateResults(results []reflect.Value) (any, error) {
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		return results[0].Interface(), nil
	case 2:
		if !results[1].IsNil() {
			err, ok := results[1].Interface().(error)
			if !ok {
				return nil, fmt.Errorf("gotmpl: helper second return value must be an error")
			}
			return nil, err
		}
		return results[0].Interface(), nil
	default:
		return nil, fmt.Errorf("gotmpl: helper returned too many values")
	}
}

func templateVariables(variables map[string]any) map[string]any {
	out := make(map[string]any, len(variables))
	for name, value := range variables {
		out[name] = templateValue(value)
	}
	return out
}

func templateValue(value any) any {
	fragment, ok := value.(lazyview.Fragment)
	if !ok {
		return value
	}
	if fragment.ContentType == "text/html; charset=utf-8" {
		return template.HTML(fragment.Body)
	}
	return fragment.Body
}
