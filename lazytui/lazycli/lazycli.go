// Package lazycli provides a small reflection-driven command router.
package lazycli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"golazy.dev/lazysupport/inflection"
	"golazy.dev/lazytui"
)

type Drawer func(Scope)

// Command configures one command scope. Empty presentation fields inherit from
// lower merged layers and, for Logo and Palette, from parent scopes.
type Command struct {
	Name         string
	Help         string
	Description  string
	Hidden       bool
	Disabled     bool
	Default      bool
	Logo         string
	Palette      lazytui.Palette
	Controller   any
	Method       string
	Autocomplete bool
}

// Scope builds a command tree. Repeated definitions overlay the existing node;
// controllers are composed in definition order.
type Scope interface {
	Root(Command)
	Define(name string, command Command)
	Group(name string, command Command, draw Drawer)
	Merge(Drawer)
}

type scope struct {
	root *node
	node *node
}

type node struct {
	name         string
	parent       *node
	help         string
	description  string
	hidden       bool
	disabled     bool
	defaultChild bool
	logo         string
	palette      lazytui.Palette
	autocomplete bool
	method       string
	controllers  []any
	children     map[string]*node
	order        []string
}

func newRoot(name string) *node {
	return &node{name: name, children: map[string]*node{}}
}

func (s *scope) Root(command Command) {
	if s.node != s.root {
		panic("lazycli: Root may only be called on the root scope")
	}
	s.node.apply(command, true)
}

func (s *scope) Define(name string, command Command) {
	child := s.child(name)
	child.apply(command, false)
}

func (s *scope) Group(name string, command Command, draw Drawer) {
	child := s.child(name)
	child.apply(command, false)
	if draw != nil {
		draw(&scope{root: s.root, node: child})
	}
}

func (s *scope) Merge(draw Drawer) {
	if draw != nil {
		draw(s)
	}
}

func (s *scope) child(name string) *node {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, " \t\r\n") || strings.HasPrefix(name, "-") {
		panic(fmt.Sprintf("lazycli: invalid command name %q", name))
	}
	if child := s.node.children[name]; child != nil {
		return child
	}
	child := &node{name: name, parent: s.node, children: map[string]*node{}}
	s.node.children[name] = child
	s.node.order = append(s.node.order, name)
	return child
}

func (n *node) apply(command Command, root bool) {
	if strings.ContainsAny(command.Help, "\r\n") {
		panic(fmt.Sprintf("lazycli: help for %q must be one line", n.name))
	}
	if root && command.Name != "" {
		n.name = command.Name
	}
	if command.Help != "" {
		n.help = command.Help
	}
	if command.Description != "" {
		n.description = strings.TrimSpace(command.Description)
	}
	if command.Hidden {
		n.hidden = true
	}
	if command.Disabled {
		n.disabled = true
	}
	if command.Default {
		if root {
			panic("lazycli: the root command cannot be its own default")
		}
		n.defaultChild = true
	}
	if command.Logo != "" {
		n.logo = command.Logo
	}
	if !command.Palette.IsZero() {
		n.palette = command.Palette
	}
	if command.Autocomplete {
		n.autocomplete = true
	}
	if command.Method != "" {
		n.method = command.Method
	}
	if command.Controller != nil {
		n.controllers = append(n.controllers, command.Controller)
	}
}

type invocationIO struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

type ioContextKey struct{}

func withIO(ctx context.Context, streams invocationIO) context.Context {
	return context.WithValue(ctx, ioContextKey{}, streams)
}

func streamsFrom(ctx context.Context) invocationIO {
	streams, _ := ctx.Value(ioContextKey{}).(invocationIO)
	return streams
}

func Stdin(ctx context.Context) io.Reader  { return streamsFrom(ctx).stdin }
func Stdout(ctx context.Context) io.Writer { return streamsFrom(ctx).stdout }
func Stderr(ctx context.Context) io.Writer { return streamsFrom(ctx).stderr }

type paramsContextKey struct{}

// Params returns the parsed parameter struct of type T for the invocation.
func Params[T any](ctx context.Context) T {
	values, _ := ctx.Value(paramsContextKey{}).(map[reflect.Type]reflect.Value)
	typeOf := reflect.TypeFor[T]()
	if value, ok := values[typeOf]; ok {
		return value.Interface().(T)
	}
	if typeOf.Kind() == reflect.Pointer {
		if value, ok := values[typeOf.Elem()]; ok {
			pointer := reflect.New(typeOf.Elem())
			pointer.Elem().Set(value)
			return pointer.Interface().(T)
		}
	}
	var zero T
	return zero
}

// Func adapts func(context.Context, Params) error into a command controller.
func Func(handler any) any {
	value := reflect.ValueOf(handler)
	if !value.IsValid() || value.Kind() != reflect.Func {
		panic("lazycli: Func requires a function")
	}
	validateAction(value.Type(), "Func")
	return functionController{handler: value}
}

type functionController struct {
	handler reflect.Value
}

// Controller is a zero-size controller type reference. It lets a router expose
// method metadata without constructing the controller during help or
// completion.
type Controller[T any] struct{}

type controllerReference interface {
	controllerType() reflect.Type
}

func (Controller[T]) controllerType() reflect.Type { return reflect.TypeFor[T]() }

func compile(draw Drawer, binaryName string) *node {
	name := filepath.Base(binaryName)
	if name == "" || name == "." {
		name = "command"
	}
	root := newRoot(name)
	if draw != nil {
		draw(&scope{root: root, node: root})
	}
	validateTree(root)
	return root
}

func validateTree(root *node) {
	var walk func(*node)
	walk = func(current *node) {
		var defaultName string
		for _, name := range current.order {
			child := current.children[name]
			if !child.defaultChild {
				continue
			}
			if defaultName != "" {
				panic(fmt.Sprintf("lazycli: commands %q and %q are both defaults under %q", defaultName, name, current.path()))
			}
			defaultName = name
		}
		for _, controller := range current.controllers {
			validateController(controller, current.name)
		}
		for _, name := range current.order {
			walk(current.children[name])
		}
		if current != root && len(current.children) == 0 && len(current.controllers) == 0 {
			method, controller := findAction(current)
			if controller == nil {
				panic(fmt.Sprintf("lazycli: command %q has no controller method %s", current.path(), method))
			}
		}
	}
	walk(root)
}

func validateController(controller any, route string) {
	if function, ok := controller.(functionController); ok {
		validateAction(function.handler.Type(), route)
		return
	}
	typeOf := controllerTypeOf(controller)
	if typeOf == nil {
		panic(fmt.Sprintf("lazycli: nil controller for %q", route))
	}
	if typeOf.Kind() != reflect.Pointer {
		typeOf = reflect.PointerTo(typeOf)
	}
	for _, method := range []string{"Before", "Run", "After", "Cleanup"} {
		if candidate, ok := typeOf.MethodByName(method); ok {
			validateMethod(candidate.Type, method, method == "Before")
		}
	}
}

func controllerTypeOf(controller any) reflect.Type {
	if reference, ok := controller.(controllerReference); ok {
		return reference.controllerType()
	}
	return reflect.TypeOf(controller)
}

func validateAction(typeOf reflect.Type, name string) {
	validateCallable(typeOf, name, false, false)
}

func validateMethod(typeOf reflect.Type, name string, before bool) {
	validateCallable(typeOf, name, true, before)
}

func validateCallable(typeOf reflect.Type, name string, method, before bool) {
	offset := 0
	if method {
		offset = 1
	}
	if typeOf.NumIn() != offset+2 || typeOf.In(offset) != reflect.TypeFor[context.Context]() {
		panic(fmt.Sprintf("lazycli: %s must accept context.Context and one params struct", name))
	}
	params := typeOf.In(offset + 1)
	if params.Kind() == reflect.Pointer {
		params = params.Elem()
	}
	if params.Kind() != reflect.Struct {
		panic(fmt.Sprintf("lazycli: %s params must be a struct", name))
	}
	errorType := reflect.TypeFor[error]()
	if before {
		if typeOf.NumOut() != 2 || typeOf.Out(0) != reflect.TypeFor[context.Context]() || typeOf.Out(1) != errorType {
			panic("lazycli: Before must return (context.Context, error)")
		}
		return
	}
	if typeOf.NumOut() != 1 || typeOf.Out(0) != errorType {
		panic(fmt.Sprintf("lazycli: %s must return error", name))
	}
}

func (n *node) path() string {
	var parts []string
	for current := n; current != nil; current = current.parent {
		parts = append(parts, current.name)
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(parts, " ")
}

func effectivePalette(n *node) lazytui.Palette {
	for current := n; current != nil; current = current.parent {
		if !current.palette.IsZero() {
			return current.palette
		}
	}
	return lazytui.Palette{}
}

func effectiveLogo(n *node) string {
	for current := n; current != nil; current = current.parent {
		if current.logo != "" {
			return current.logo
		}
	}
	return ""
}

func sortedChildren(n *node) []*node {
	children := make([]*node, 0, len(n.children))
	for _, child := range n.children {
		if !child.hidden && !child.disabled {
			children = append(children, child)
		}
	}
	sort.SliceStable(children, func(i, j int) bool { return children[i].name < children[j].name })
	return children
}

func actionName(n *node) string {
	if n.method != "" {
		return n.method
	}
	return inflection.Camelize(n.name)
}
