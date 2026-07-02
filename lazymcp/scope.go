package lazymcp

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
)

// Options configures a Scope.
type Options struct {
	Views      Views
	Authorizer func(context.Context, string) bool
}

// Base is embedded by application MCP module types.
type Base struct {
	ctx  context.Context
	name string
}

// NewBase creates an MCP base value.
func NewBase(ctx context.Context) Base {
	return Base{ctx: ctx}
}

// Name returns the explicit module name. Empty means infer from type/package.
func (base Base) Name() string {
	return base.name
}

// Context returns the application context captured by NewBase.
func (base Base) Context() context.Context {
	return base.ctx
}

// Scope owns registered MCP modules and serves the MCP HTTP endpoint.
type Scope struct {
	options Options
	modules map[string]*module
}

type module struct {
	name      string
	component any
	tools     map[string]ToolSpec
	resources map[string]ResourceSpec
	prompts   map[string]PromptSpec
	skills    map[string]SkillSpec
	apps      map[string]AppSpec
}

// NewScope creates an empty MCP scope.
func NewScope(options Options) *Scope {
	return &Scope{options: options, modules: map[string]*module{}}
}

// Register registers one MCP module component.
func (scope *Scope) Register(component any) error {
	if scope == nil {
		return fmt.Errorf("lazymcp: scope is nil")
	}
	if component == nil {
		return fmt.Errorf("lazymcp: component is nil")
	}
	name := componentName(component)
	if name == "" {
		return fmt.Errorf("lazymcp: component name is required")
	}
	if _, exists := scope.modules[name]; exists {
		return fmt.Errorf("lazymcp: component %q is already registered", name)
	}
	m := &module{
		name:      name,
		component: component,
		tools:     map[string]ToolSpec{},
		resources: map[string]ResourceSpec{},
		prompts:   map[string]PromptSpec{},
		skills:    map[string]SkillSpec{},
		apps:      map[string]AppSpec{},
	}
	if err := m.reflectSpecs(contextFromComponent(component)); err != nil {
		return err
	}
	scope.modules[name] = m
	return nil
}

// Empty reports whether the scope has no registered modules.
func (scope *Scope) Empty() bool {
	return scope == nil || len(scope.modules) == 0
}

// MiddlewareName returns the dispatcher-visible middleware name.
func (scope *Scope) MiddlewareName() string {
	return "lazymcp.Scope"
}

// Handler serves MCP requests at /mcp and falls through to next for other paths.
func (scope *Scope) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			next.ServeHTTP(w, r)
			return
		}
		scope.ServeHTTP(w, r)
	})
}

func (m *module) reflectSpecs(ctx context.Context) error {
	value := reflect.ValueOf(m.component)
	typ := value.Type()
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		if method.PkgPath != "" {
			continue
		}
		switch {
		case strings.HasSuffix(method.Name, "Tool"):
			spec, err := callSpec[ToolSpec](ctx, value.Method(i))
			if err != nil {
				return fmt.Errorf("%s.%s: %w", typ, method.Name, err)
			}
			if spec.Name == "" {
				spec.Name = inferredSpecName(method.Name, "Tool")
			}
			m.tools[spec.Name] = spec
		case strings.HasSuffix(method.Name, "Resource"):
			spec, err := callSpec[ResourceSpec](ctx, value.Method(i))
			if err != nil {
				return fmt.Errorf("%s.%s: %w", typ, method.Name, err)
			}
			if spec.Name == "" {
				spec.Name = inferredSpecName(method.Name, "Resource")
			}
			if spec.URI == "" {
				spec.URI = "app://" + m.name + "/" + spec.Name
			}
			m.resources[spec.URI] = spec
		case strings.HasSuffix(method.Name, "Prompt"):
			spec, err := callSpec[PromptSpec](ctx, value.Method(i))
			if err != nil {
				return fmt.Errorf("%s.%s: %w", typ, method.Name, err)
			}
			if spec.Name == "" {
				spec.Name = inferredSpecName(method.Name, "Prompt")
			}
			m.prompts[spec.Name] = spec
		case strings.HasSuffix(method.Name, "Skill"):
			spec, err := callSpec[SkillSpec](ctx, value.Method(i))
			if err != nil {
				return fmt.Errorf("%s.%s: %w", typ, method.Name, err)
			}
			if spec.Path == "" {
				spec.Path = inferredSpecName(method.Name, "Skill")
			}
			m.skills[spec.Path] = spec
		case strings.HasSuffix(method.Name, "App"):
			spec, err := callSpec[AppSpec](ctx, value.Method(i))
			if err != nil {
				return fmt.Errorf("%s.%s: %w", typ, method.Name, err)
			}
			if spec.Name == "" {
				spec.Name = inferredSpecName(method.Name, "App")
			}
			if spec.URI == "" {
				spec.URI = "ui://" + m.name + "/" + spec.Name
			}
			m.apps[spec.URI] = spec
		}
	}
	addAggregate(ctx, value.MethodByName("Tools"), m.tools)
	addAggregate(ctx, value.MethodByName("Resources"), m.resources)
	addAggregate(ctx, value.MethodByName("Prompts"), m.prompts)
	addAggregate(ctx, value.MethodByName("Skills"), m.skills)
	addAggregate(ctx, value.MethodByName("Apps"), m.apps)
	return nil
}

func callSpec[T any](ctx context.Context, method reflect.Value) (T, error) {
	var zero T
	if !method.IsValid() {
		return zero, fmt.Errorf("method is missing")
	}
	args := []reflect.Value{}
	if method.Type().NumIn() == 1 {
		args = append(args, reflect.ValueOf(ctx))
	}
	out := method.Call(args)
	if len(out) == 0 {
		return zero, fmt.Errorf("method returned no value")
	}
	if len(out) == 2 && !out[1].IsNil() {
		err, _ := out[1].Interface().(error)
		return zero, err
	}
	spec, ok := out[0].Interface().(T)
	if !ok {
		return zero, fmt.Errorf("unexpected return type %T", out[0].Interface())
	}
	return spec, nil
}

func addAggregate[T any](ctx context.Context, method reflect.Value, target map[string]T) {
	if !method.IsValid() {
		return
	}
	args := []reflect.Value{}
	if method.Type().NumIn() == 1 {
		args = append(args, reflect.ValueOf(ctx))
	}
	out := method.Call(args)
	if len(out) == 0 {
		return
	}
	items, ok := out[0].Interface().([]T)
	if !ok {
		return
	}
	for _, item := range items {
		name := aggregateName(item)
		if name != "" {
			target[name] = item
		}
	}
}

func aggregateName(item any) string {
	switch spec := item.(type) {
	case ToolSpec:
		return spec.Name
	case ResourceSpec:
		return firstNonEmpty(spec.URI, spec.Name)
	case PromptSpec:
		return spec.Name
	case SkillSpec:
		return spec.Path
	case AppSpec:
		return firstNonEmpty(spec.URI, spec.Name)
	default:
		return ""
	}
}

func componentName(component any) string {
	if named, ok := component.(interface{ Name() string }); ok {
		if name := normalizeName(named.Name()); name != "" {
			return name
		}
	}
	typ := reflect.TypeOf(component)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return normalizeName(typ.Name())
}

func contextFromComponent(component any) context.Context {
	if contextual, ok := component.(interface{ Context() context.Context }); ok {
		if ctx := contextual.Context(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func inferredSpecName(methodName string, suffix string) string {
	return normalizeName(strings.TrimSuffix(methodName, suffix))
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimSuffix(name, "MCP")
	name = strings.TrimSuffix(name, "_mcp")
	name = strings.TrimSuffix(name, "-mcp")
	name = camelToSnake(name)
	name = strings.Trim(name, "_-")
	return strings.ReplaceAll(name, "-", "_")
}

var camelBoundary = regexp.MustCompile(`([a-z0-9])([A-Z])`)

func camelToSnake(value string) string {
	value = camelBoundary.ReplaceAllString(value, `${1}_${2}`)
	return strings.ToLower(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
