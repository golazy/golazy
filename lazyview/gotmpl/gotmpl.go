// Package gotmpl registers Go's html/template engine for lazyview.
package gotmpl

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"path"
	"reflect"
	"strings"
	"sync"

	"golazy.dev/lazyview"
)

func init() {
	lazyview.RegisterEngine("tpl", func() lazyview.Engine {
		return &Engine{}
	})
}

// Engine renders .tpl files with html/template.
type Engine struct {
	mu        sync.RWMutex
	templates map[string]*cachedTemplate
}

// Cache parses all templates for this engine's extension and replaces the cache.
func (e *Engine) Cache(ctx lazyview.CacheContext) error {
	templates := map[string]*cachedTemplate{}
	extension := "." + strings.TrimPrefix(ctx.Extension, ".")
	helpers := copyHelpers(ctx.Helpers)

	err := fs.WalkDir(ctx.FS, ".", func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || path.Ext(file) != extension {
			return nil
		}

		data, err := fs.ReadFile(ctx.FS, file)
		if err != nil {
			return err
		}
		tpl, err := newCachedTemplate(file, string(data), helpers)
		if err != nil {
			return err
		}
		templates[file] = tpl
		return nil
	})
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.templates = templates
	e.mu.Unlock()
	return nil
}

// ClearCache discards parsed templates.
func (e *Engine) ClearCache() {
	e.mu.Lock()
	e.templates = nil
	e.mu.Unlock()
}

// Render renders one template file.
func (e *Engine) Render(ctx *lazyview.Context, writer io.Writer, file string) error {
	if tpl, ok := e.cachedTemplate(file); ok {
		return tpl.Execute(ctx, writer)
	}
	tpl, err := parseTemplate(ctx.Views.FS, file, templateHelpers(ctx.HelperFuncs()))
	if err != nil {
		return err
	}
	return tpl.Execute(writer, templateData(ctx))
}

func (e *Engine) cachedTemplate(file string) (*cachedTemplate, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.templates) == 0 {
		return nil, false
	}
	tpl, ok := e.templates[file]
	return tpl, ok
}

type cachedTemplate struct {
	file    string
	source  string
	helpers map[string]any
	pool    sync.Pool
}

type templateExecutor struct {
	ctx *lazyview.Context
	tpl *template.Template
}

func newCachedTemplate(file string, source string, helpers map[string]any) (*cachedTemplate, error) {
	cached := &cachedTemplate{
		file:    file,
		source:  source,
		helpers: helpers,
	}
	executor, err := cached.newExecutor()
	if err != nil {
		return nil, err
	}
	cached.pool.Put(executor)
	return cached, nil
}

func (c *cachedTemplate) Execute(ctx *lazyview.Context, writer io.Writer) error {
	executor, err := c.executor()
	if err != nil {
		return err
	}
	executor.ctx = ctx
	err = executor.tpl.Execute(writer, templateData(ctx))
	executor.ctx = nil
	c.pool.Put(executor)
	return err
}

func (c *cachedTemplate) executor() (*templateExecutor, error) {
	value := c.pool.Get()
	if value == nil {
		return c.newExecutor()
	}
	return value.(*templateExecutor), nil
}

func (c *cachedTemplate) newExecutor() (*templateExecutor, error) {
	executor := &templateExecutor{}
	tpl, err := template.New(c.file).
		Funcs(executor.templateHelpers(c.helpers)).
		Parse(c.source)
	if err != nil {
		return nil, err
	}
	executor.tpl = tpl
	return executor, nil
}

func (e *templateExecutor) templateHelpers(helpers map[string]any) template.FuncMap {
	funcs := template.FuncMap{}
	for name, helper := range helpers {
		value := reflect.ValueOf(helper)
		if !value.IsValid() || value.Kind() != reflect.Func {
			continue
		}
		helperName := name
		funcs[helperName] = func(args ...any) (any, error) {
			return e.callHelper(helperName, args)
		}
	}
	return funcs
}

func (e *templateExecutor) callHelper(name string, args []any) (any, error) {
	if e.ctx == nil {
		return nil, fmt.Errorf("gotmpl: helper %q called outside render", name)
	}
	helper, ok := e.ctx.Helper(name)
	if !ok {
		return nil, fmt.Errorf("gotmpl: helper %q is missing", name)
	}
	value := reflect.ValueOf(helper)
	if !value.IsValid() || value.Kind() != reflect.Func {
		return nil, fmt.Errorf("gotmpl: helper %q is not a function", name)
	}
	result, err := templateHelper(value)(args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func parseTemplate(files fs.FS, file string, helpers template.FuncMap) (*template.Template, error) {
	data, err := fs.ReadFile(files, file)
	if err != nil {
		return nil, err
	}
	return template.New(file).Funcs(helpers).Parse(string(data))
}

func copyHelpers(helpers map[string]any) map[string]any {
	if len(helpers) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(helpers))
	maps.Copy(out, helpers)
	return out
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
	if !hasTemplateFragments(variables) {
		return variables
	}
	out := make(map[string]any, len(variables))
	for name, value := range variables {
		out[name] = templateValue(value)
	}
	return out
}

func hasTemplateFragments(variables map[string]any) bool {
	for _, value := range variables {
		if _, ok := value.(lazyview.Fragment); ok {
			return true
		}
	}
	return false
}

func templateData(ctx *lazyview.Context) any {
	if ctx.Data == nil {
		return templateVariables(ctx.Variables)
	}
	if variables, ok := ctx.Data.(map[string]any); ok {
		return templateVariables(variables)
	}
	return templateValue(ctx.Data)
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
