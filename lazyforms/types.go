package lazyforms

import (
	"fmt"
	"reflect"
	"strings"

	"golazy.dev/lazyschema"
)

type Resource interface {
	Persisted() bool
	RouteParam() string
}

type NumericID interface {
	ID() int
}

type StringID interface {
	ID() string
}

type Router interface {
	PathForModel(model any, action string) (string, error)
}

type routeRouter interface {
	PathFor(name string, values ...any) (string, error)
}

type Form struct {
	Model       any
	ModelType   reflect.Type
	ModelKey    string
	Action      string
	Method      string
	HTMLMethod  string
	ID          string
	Class       string
	Partial     string
	Multipart   bool
	Attrs       map[string]string
	Data        map[string]string
	Fields      []Field
	OriginalDot any
}

type Field struct {
	Form        *Form
	GoPath      string
	Path        lazyschema.Path
	Name        string
	ID          string
	Label       string
	Type        string
	Value       any
	Attrs       map[string]string
	Placeholder string
}

type formOptions struct {
	action      string
	routeName   string
	routeValues []any
	method      string
	id          string
	class       string
	addClasses  []string
	partial     string
	modelName   string
	multipart   bool
	attrs       map[string]string
	data        map[string]string
}

type Option func(*formOptions)

func FormAction(action string) Option {
	return func(options *formOptions) {
		options.action = action
	}
}

func FormRoute(name string, values ...any) Option {
	return func(options *formOptions) {
		options.routeName = name
		options.routeValues = append([]any(nil), values...)
	}
}

func FormMethod(method string) Option {
	return func(options *formOptions) {
		options.method = strings.ToLower(method)
	}
}

func FormID(id string) Option {
	return func(options *formOptions) {
		options.id = id
	}
}

func FormClass(class string) Option {
	return func(options *formOptions) {
		options.class = class
	}
}

func FormAddClass(class string) Option {
	return func(options *formOptions) {
		if strings.TrimSpace(class) != "" {
			options.addClasses = append(options.addClasses, class)
		}
	}
}

func FormFile(partial string) Option {
	return func(options *formOptions) {
		options.partial = partial
	}
}

func FormModel(modelName string) Option {
	return func(options *formOptions) {
		options.modelName = modelName
	}
}

func FormScope(scope string) Option {
	return FormModel(scope)
}

func FormMultipart() Option {
	return func(options *formOptions) {
		options.multipart = true
	}
}

func FormAttr(name string, value any) Option {
	return func(options *formOptions) {
		if options.attrs == nil {
			options.attrs = map[string]string{}
		}
		options.attrs[name] = fmt.Sprint(value)
	}
}

func FormData(name string, value any) Option {
	return func(options *formOptions) {
		if options.data == nil {
			options.data = map[string]string{}
		}
		options.data[name] = fmt.Sprint(value)
	}
}

type fieldOptions struct {
	label       string
	inputType   string
	id          string
	class       string
	value       any
	placeholder string
	attrs       map[string]string
	data        map[string]string
}

type FieldOption func(*fieldOptions)

func FieldLabel(label string) FieldOption {
	return func(options *fieldOptions) {
		options.label = label
	}
}

func FieldType(inputType string) FieldOption {
	return func(options *fieldOptions) {
		options.inputType = inputType
	}
}

func FieldID(id string) FieldOption {
	return func(options *fieldOptions) {
		options.id = id
	}
}

func FieldClass(class string) FieldOption {
	return func(options *fieldOptions) {
		options.class = class
	}
}

func FieldValue(value any) FieldOption {
	return func(options *fieldOptions) {
		options.value = value
	}
}

func FieldPlaceholder(value string) FieldOption {
	return func(options *fieldOptions) {
		options.placeholder = value
	}
}

func FieldAttr(name string, value any) FieldOption {
	return func(options *fieldOptions) {
		if options.attrs == nil {
			options.attrs = map[string]string{}
		}
		options.attrs[name] = fmt.Sprint(value)
	}
}

func FieldData(name string, value any) FieldOption {
	return func(options *fieldOptions) {
		if options.data == nil {
			options.data = map[string]string{}
		}
		options.data[name] = fmt.Sprint(value)
	}
}
