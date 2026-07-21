package lazycli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"golazy.dev/lazysupport/inflection"
)

type ExecuteConfig struct {
	Args   []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	Name   string
}

// Main executes a drawer using the process arguments and streams.
func Main(draw Drawer) {
	os.Exit(Execute(context.Background(), draw, ExecuteConfig{
		Args: os.Args[1:], Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, Name: os.Args[0],
	}))
}

// Execute draws and runs a command tree. It returns a process-style exit code.
func Execute(ctx context.Context, draw Drawer, config ExecuteConfig) int {
	streams := invocationIO{stdin: config.Stdin, stdout: config.Stdout, stderr: config.Stderr}
	if streams.stdin == nil {
		streams.stdin = strings.NewReader("")
	}
	if streams.stdout == nil {
		streams.stdout = io.Discard
	}
	if streams.stderr == nil {
		streams.stderr = io.Discard
	}
	name := config.Name
	if name == "" {
		name = "command"
	}
	root := compile(draw, name)
	if len(config.Args) > 0 && config.Args[0] == "__complete" && root.autocomplete {
		return executeComplete(root, config.Args[1:], streams.stdout)
	}
	if len(config.Args) > 0 && config.Args[0] == "completion" && root.autocomplete {
		return executeCompletionScript(root, config.Args[1:], streams.stdout, streams.stderr)
	}

	parsed, err := parse(root, config.Args)
	if err != nil {
		writeError(streams.stderr, effectivePalette(root), err)
		return 2
	}
	if parsed.help {
		writeHelp(streams.stdout, parsed.selected)
		return 0
	}
	if parsed.logo {
		logo := effectiveLogo(parsed.selected)
		if logo == "" {
			writeError(streams.stderr, effectivePalette(parsed.selected), errors.New("logo is not configured"))
			return 2
		}
		fmt.Fprintln(streams.stdout, colorize(streams.stdout, effectivePalette(parsed.selected).Primary, logo))
		return 0
	}
	if parsed.selected.disabled {
		writeError(streams.stderr, effectivePalette(parsed.selected), fmt.Errorf("command %q is disabled", parsed.selected.path()))
		return 2
	}

	plan, err := buildPlan(parsed.selected)
	if err != nil {
		if len(parsed.selected.children) > 0 {
			writeHelp(streams.stdout, parsed.selected)
			return 0
		}
		writeError(streams.stderr, effectivePalette(parsed.selected), err)
		return 2
	}
	values, err := bindParams(plan.paramTypes(), parsed.flags, parsed.positionals)
	if err != nil {
		writeError(streams.stderr, effectivePalette(parsed.selected), err)
		return 2
	}
	ctx = withIO(ctx, streams)
	ctx = context.WithValue(ctx, paramsContextKey{}, values)
	if err := plan.run(ctx, values); err != nil {
		var exit exitError
		if errors.As(err, &exit) {
			return exit.code
		}
		writeError(streams.stderr, effectivePalette(parsed.selected), err)
		return 1
	}
	return 0
}

type exitError struct{ code int }

func (err exitError) Error() string { return fmt.Sprintf("exit code %d", err.code) }

// Exit returns an error that stops command execution with code without
// printing another diagnostic. A zero code returns nil.
func Exit(code int) error {
	if code == 0 {
		return nil
	}
	return exitError{code: code}
}

type parseResult struct {
	selected    *node
	flags       map[string][]string
	positionals []string
	help        bool
	logo        bool
}

func parse(root *node, args []string) (parseResult, error) {
	known := collectFlagNames(root)
	result := parseResult{selected: root, flags: map[string][]string{}}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "--" {
			result.positionals = append(result.positionals, args[index+1:]...)
			break
		}
		if arg == "-h" || arg == "--help" {
			result.help = true
			continue
		}
		if arg == "--logo" {
			result.logo = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			name, value, hasValue := splitFlag(arg)
			spec, ok := known[name]
			if !ok {
				return result, fmt.Errorf("unknown parameter %q", arg)
			}
			if !hasValue && spec.kind != reflect.Bool {
				if index+1 >= len(args) {
					return result, fmt.Errorf("parameter --%s requires a value", spec.name)
				}
				index++
				value = args[index]
				hasValue = true
			}
			if !hasValue {
				value = "true"
			}
			result.flags[spec.name] = append(result.flags[spec.name], value)
			continue
		}
		if child := result.selected.children[arg]; child != nil && !child.disabled && len(result.positionals) == 0 {
			result.selected = child
			continue
		}
		result.positionals = append(result.positionals, arg)
	}
	if !result.help && !result.logo && len(result.positionals) == 0 {
		for _, name := range result.selected.order {
			child := result.selected.children[name]
			if child.defaultChild && !child.disabled {
				result.selected = child
				break
			}
		}
	}
	return result, nil
}

func splitFlag(arg string) (string, string, bool) {
	trimmed := strings.TrimLeft(arg, "-")
	if name, value, ok := strings.Cut(trimmed, "="); ok {
		return name, value, true
	}
	return trimmed, "", false
}

type executionPlan struct {
	nodes  []controllerCall
	action callable
}

type controllerCall struct {
	value   reflect.Value
	before  *callable
	after   *callable
	cleanup *callable
}

type callable struct {
	function reflect.Value
	params   reflect.Type
	before   bool
}

func buildPlan(selected *node) (*executionPlan, error) {
	var path []*node
	for current := selected; current != nil; current = current.parent {
		path = append(path, current)
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	plan := &executionPlan{}
	instances := map[controllerIdentity]reflect.Value{}
	for _, current := range path {
		for _, definition := range current.controllers {
			value := instantiate(definition)
			instances[identity(definition)] = value
			call := controllerCall{value: value}
			call.before = methodCallable(value, "Before", true)
			call.after = methodCallable(value, "After", false)
			call.cleanup = methodCallable(value, "Cleanup", false)
			plan.nodes = append(plan.nodes, call)
		}
	}

	if action, definition := findAction(selected); definition != nil {
		value := instances[identity(definition)]
		if function, ok := definition.(functionController); ok {
			plan.action = callable{function: function.handler, params: function.handler.Type().In(1)}
			return plan, nil
		}
		method := value.MethodByName(action)
		plan.action = callable{function: method, params: method.Type().In(1)}
		return plan, nil
	}
	return nil, fmt.Errorf("command %q has no action", selected.path())
}

type controllerIdentity struct {
	typeOf reflect.Type
	index  uintptr
}

func identity(definition any) controllerIdentity {
	if function, ok := definition.(functionController); ok {
		return controllerIdentity{typeOf: reflect.TypeOf(function), index: function.handler.Pointer()}
	}
	return controllerIdentity{typeOf: controllerTypeOf(definition)}
}

func instantiate(definition any) reflect.Value {
	if function, ok := definition.(functionController); ok {
		return reflect.ValueOf(function)
	}
	typeOf := controllerTypeOf(definition)
	if typeOf.Kind() == reflect.Pointer {
		return reflect.New(typeOf.Elem())
	}
	value := reflect.New(typeOf)
	return value
}

func methodCallable(value reflect.Value, name string, before bool) *callable {
	method := value.MethodByName(name)
	if !method.IsValid() {
		return nil
	}
	return &callable{function: method, params: method.Type().In(1), before: before}
}

func findAction(selected *node) (string, any) {
	if len(selected.controllers) > 0 {
		for index := len(selected.controllers) - 1; index >= 0; index-- {
			definition := selected.controllers[index]
			if _, ok := definition.(functionController); ok {
				return "Run", definition
			}
			typeOf := controllerTypeOf(definition)
			if typeOf.Kind() != reflect.Pointer {
				typeOf = reflect.PointerTo(typeOf)
			}
			if _, ok := typeOf.MethodByName("Run"); ok {
				return "Run", definition
			}
		}
	}
	method := actionName(selected)
	for current := selected.parent; current != nil; current = current.parent {
		for index := len(current.controllers) - 1; index >= 0; index-- {
			definition := current.controllers[index]
			if _, ok := definition.(functionController); ok {
				if method == "Run" {
					return method, definition
				}
				continue
			}
			typeOf := controllerTypeOf(definition)
			if typeOf.Kind() != reflect.Pointer {
				typeOf = reflect.PointerTo(typeOf)
			}
			if _, ok := typeOf.MethodByName(method); ok {
				return method, definition
			}
		}
	}
	return method, nil
}

func (plan *executionPlan) paramTypes() []reflect.Type {
	seen := map[reflect.Type]bool{}
	var types []reflect.Type
	add := func(call *callable) {
		if call == nil {
			return
		}
		typeOf := call.params
		if typeOf.Kind() == reflect.Pointer {
			typeOf = typeOf.Elem()
		}
		if !seen[typeOf] {
			seen[typeOf] = true
			types = append(types, typeOf)
		}
	}
	for index := range plan.nodes {
		add(plan.nodes[index].before)
		add(plan.nodes[index].after)
		add(plan.nodes[index].cleanup)
	}
	add(&plan.action)
	return types
}

func (plan *executionPlan) run(ctx context.Context, values map[reflect.Type]reflect.Value) (err error) {
	entered := make([]controllerCall, 0, len(plan.nodes))
	for _, node := range plan.nodes {
		entered = append(entered, node)
		if node.before == nil {
			continue
		}
		results := node.before.function.Call([]reflect.Value{reflect.ValueOf(ctx), paramValue(values, node.before.params)})
		if !results[0].IsNil() {
			ctx = results[0].Interface().(context.Context)
		}
		if callErr := errorResult(results[1]); callErr != nil {
			err = callErr
			break
		}
	}
	if err == nil {
		results := plan.action.function.Call([]reflect.Value{reflect.ValueOf(ctx), paramValue(values, plan.action.params)})
		err = errorResult(results[0])
	}
	if err == nil {
		for index := len(entered) - 1; index >= 0; index-- {
			if entered[index].after == nil {
				continue
			}
			results := entered[index].after.function.Call([]reflect.Value{reflect.ValueOf(ctx), paramValue(values, entered[index].after.params)})
			if afterErr := errorResult(results[0]); afterErr != nil {
				err = errors.Join(err, afterErr)
			}
		}
	}
	for index := len(entered) - 1; index >= 0; index-- {
		if entered[index].cleanup == nil {
			continue
		}
		results := entered[index].cleanup.function.Call([]reflect.Value{reflect.ValueOf(ctx), paramValue(values, entered[index].cleanup.params)})
		if cleanupErr := errorResult(results[0]); cleanupErr != nil {
			err = errors.Join(err, cleanupErr)
		}
	}
	return err
}

func paramValue(values map[reflect.Type]reflect.Value, wanted reflect.Type) reflect.Value {
	pointer := wanted.Kind() == reflect.Pointer
	key := wanted
	if pointer {
		key = wanted.Elem()
	}
	value := values[key]
	if !pointer {
		return value
	}
	result := reflect.New(key)
	result.Elem().Set(value)
	return result
}

func errorResult(value reflect.Value) error {
	if !value.IsValid() || value.IsNil() {
		return nil
	}
	return value.Interface().(error)
}

type flagSpec struct {
	name         string
	short        string
	help         string
	defaultValue string
	kind         reflect.Kind
	enum         []string
	field        int
	typeOf       reflect.Type
	positional   bool
	variadic     bool
	repeatable   bool
}

func specsFor(typeOf reflect.Type) []flagSpec {
	if typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	var specs []flagSpec
	for index := 0; index < typeOf.NumField(); index++ {
		field := typeOf.Field(index)
		if !field.IsExported() {
			continue
		}
		spec := flagSpec{name: inflection.Dasherize(field.Name), kind: field.Type.Kind(), field: index, typeOf: typeOf}
		if field.Type.Kind() == reflect.Slice {
			spec.kind = field.Type.Elem().Kind()
			spec.repeatable = true
		}
		for _, setting := range strings.Split(field.Tag.Get("lazycli"), ",") {
			key, value, hasValue := strings.Cut(strings.TrimSpace(setting), "=")
			switch key {
			case "":
			case "name":
				spec.name = value
			case "short":
				spec.short = value
			case "default":
				spec.defaultValue = value
			case "help":
				spec.help = value
			case "enum":
				spec.enum = strings.Split(value, "|")
			case "positional":
				spec.positional = true
			case "variadic":
				spec.positional, spec.variadic = true, true
			case "repeatable":
				spec.repeatable = true
			default:
				if hasValue {
					panic(fmt.Sprintf("lazycli: unknown field setting %q", key))
				}
			}
		}
		if spec.help == "" {
			spec.help = field.Tag.Get("help")
		}
		specs = append(specs, spec)
	}
	return specs
}

func bindParams(types []reflect.Type, flags map[string][]string, positionals []string) (map[reflect.Type]reflect.Value, error) {
	values := map[reflect.Type]reflect.Value{}
	known := map[string]bool{}
	position := 0
	for _, typeOf := range types {
		value := reflect.New(typeOf).Elem()
		for _, spec := range specsFor(typeOf) {
			field := value.Field(spec.field)
			if spec.positional {
				if spec.variadic {
					slice := reflect.MakeSlice(field.Type(), 0, len(positionals)-position)
					for ; position < len(positionals); position++ {
						item := reflect.New(field.Type().Elem()).Elem()
						if err := setScalar(item, positionals[position]); err != nil {
							return nil, err
						}
						slice = reflect.Append(slice, item)
					}
					field.Set(slice)
				} else if position < len(positionals) {
					if err := setScalar(field, positionals[position]); err != nil {
						return nil, err
					}
					position++
				}
				continue
			}
			known[spec.name] = true
			supplied := flags[spec.name]
			if field.Kind() == reflect.Slice {
				slice := reflect.MakeSlice(field.Type(), 0, len(supplied))
				for _, raw := range supplied {
					item := reflect.New(field.Type().Elem()).Elem()
					if err := setScalar(item, raw); err != nil {
						return nil, fmt.Errorf("--%s: %w", spec.name, err)
					}
					slice = reflect.Append(slice, item)
				}
				field.Set(slice)
				continue
			}
			raw := spec.defaultValue
			if len(supplied) > 0 {
				raw = supplied[len(supplied)-1]
			}
			if raw != "" {
				if len(spec.enum) > 0 && !contains(spec.enum, raw) {
					return nil, fmt.Errorf("--%s must be one of %s", spec.name, strings.Join(spec.enum, ", "))
				}
				if err := setScalar(field, raw); err != nil {
					return nil, fmt.Errorf("--%s: %w", spec.name, err)
				}
			}
		}
		values[typeOf] = value
	}
	for name := range flags {
		if !known[name] {
			return nil, fmt.Errorf("parameter --%s does not apply to this command", name)
		}
	}
	if position < len(positionals) {
		return nil, fmt.Errorf("unexpected argument %q", positionals[position])
	}
	return values, nil
}

func setScalar(field reflect.Value, raw string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(value)
	default:
		return fmt.Errorf("unsupported parameter type %s", field.Type())
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func collectFlagNames(root *node) map[string]flagSpec {
	result := map[string]flagSpec{}
	var visitType func(reflect.Type)
	visitType = func(typeOf reflect.Type) {
		if typeOf.Kind() == reflect.Pointer {
			typeOf = typeOf.Elem()
		}
		for _, spec := range specsFor(typeOf) {
			if spec.positional {
				continue
			}
			if previous, ok := result[spec.name]; ok && previous.kind != spec.kind {
				panic(fmt.Sprintf("lazycli: parameter --%s has conflicting types", spec.name))
			}
			result[spec.name] = spec
			if spec.short != "" {
				result[spec.short] = spec
			}
		}
	}
	var walk func(*node)
	walk = func(current *node) {
		for _, definition := range current.controllers {
			if function, ok := definition.(functionController); ok {
				visitType(function.handler.Type().In(1))
				continue
			}
			typeOf := controllerTypeOf(definition)
			if typeOf.Kind() != reflect.Pointer {
				typeOf = reflect.PointerTo(typeOf)
			}
			for index := 0; index < typeOf.NumMethod(); index++ {
				method := typeOf.Method(index)
				if method.Type.NumIn() == 3 && method.Type.In(1) == reflect.TypeFor[context.Context]() {
					visitType(method.Type.In(2))
				}
			}
		}
		for _, name := range current.order {
			walk(current.children[name])
		}
	}
	walk(root)
	return result
}
