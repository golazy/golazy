package lazyforms

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"

	"golazy.dev/lazyschema"
	"golazy.dev/lazysupport/inflection"
)

func buildForm(router Router, model any, originalDot any, options formOptions) (*Form, error) {
	form, err := modelFormDefaults(router, model, &options)
	if err != nil {
		return nil, err
	}
	form.OriginalDot = originalDot
	form.Fields = collectFields(form, form.ModelType, "", nil)
	return form, nil
}

func modelFormDefaults(router Router, model any, options *formOptions) (*Form, error) {
	if options == nil {
		options = &formOptions{}
	}
	modelType, err := modelType(model)
	if err != nil {
		if options.modelName == "" {
			return nil, err
		}
	}
	modelKey := options.modelName
	if modelKey == "" {
		modelKey, err = lazyschema.ModelNameForType(modelType)
		if err != nil {
			return nil, err
		}
	}
	htmlMethod, override := methodFor(model, options.method)
	action := options.action
	if action == "" && options.routeName != "" {
		if routeRouter, ok := router.(routeRouter); ok {
			action, err = routeRouter.PathFor(options.routeName, options.routeValues...)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("lazyforms: form_route requires a router with PathFor")
		}
	}
	if action == "" && router != nil {
		routeAction := "create"
		if override == "delete" {
			routeAction = "delete"
		} else if override != "" {
			routeAction = "update"
		}
		if path, err := router.PathForModel(model, routeAction); err == nil {
			action = path
		}
	}
	if action == "" {
		action = "#"
	}

	form := &Form{
		Model:      model,
		ModelType:  modelType,
		ModelKey:   modelKey,
		Action:     action,
		Method:     override,
		HTMLMethod: htmlMethod,
		ID:         formID(model, modelKey),
		Class:      formClass(model, modelKey),
		Partial:    partialName(modelKey),
		Multipart:  options.multipart,
		Attrs:      copyStringMap(options.attrs),
		Data:       copyStringMap(options.data),
	}
	if options.id != "" {
		form.ID = options.id
	}
	if options.class != "" {
		form.Class = options.class
	}
	form.Class = appendClasses(form.Class, options.addClasses...)
	if options.partial != "" {
		form.Partial = strings.TrimPrefix(options.partial, "_")
	}
	return form, nil
}

func modelType(model any) (reflect.Type, error) {
	if model == nil {
		return nil, fmt.Errorf("lazyforms: model is nil")
	}
	t := reflect.TypeOf(model)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("lazyforms: model must be a struct or pointer to struct")
	}
	return t, nil
}

func formID(model any, modelKey string) string {
	if isPersisted(model) {
		if id, ok := modelID(model); ok {
			return "edit_" + modelKey + "_" + id
		}
		return "edit_" + modelKey
	}
	return "new_" + modelKey
}

func formClass(model any, modelKey string) string {
	if isPersisted(model) {
		return "edit_" + modelKey
	}
	return "new_" + modelKey
}

func partialName(modelKey string) string {
	return inflection.Underscorize(modelKey) + "_form"
}

func appendClasses(base string, classes ...string) string {
	var values []string
	if strings.TrimSpace(base) != "" {
		values = append(values, strings.Fields(base)...)
	}
	for _, class := range classes {
		values = append(values, strings.Fields(class)...)
	}
	return strings.Join(values, " ")
}

func isPersisted(model any) bool {
	if model == nil || isNil(model) {
		return false
	}
	resource, ok := model.(Resource)
	return ok && resource.Persisted()
}

func modelID(model any) (string, bool) {
	if resource, ok := model.(Resource); ok {
		if param := resource.RouteParam(); param != "" {
			return param, true
		}
	}
	if id, ok := model.(NumericID); ok {
		return fmt.Sprint(id.ID()), true
	}
	if id, ok := model.(StringID); ok {
		return id.ID(), true
	}
	return "", false
}

func copyStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	out := make(map[string]string, len(source))
	maps.Copy(out, source)
	return out
}

func renderForm(form *Form, body string) string {
	var builder strings.Builder
	builder.WriteString(`<form action="`)
	builder.WriteString(attr(form.Action))
	builder.WriteString(`" method="`)
	builder.WriteString(attr(form.HTMLMethod))
	builder.WriteByte('"')
	writeAttr(&builder, "id", form.ID)
	writeAttr(&builder, "class", form.Class)
	if form.Multipart {
		writeAttr(&builder, "enctype", "multipart/form-data")
	}
	writeAttrs(&builder, form.Attrs)
	writeDataAttrs(&builder, form.Data)
	builder.WriteByte('>')
	if form.Method != "" {
		builder.WriteString(`<input type="hidden" name="_method" value="`)
		builder.WriteString(attr(form.Method))
		builder.WriteString(`">`)
	}
	builder.WriteString(body)
	builder.WriteString(`</form>`)
	return builder.String()
}

func writeAttrs(builder *strings.Builder, attrs map[string]string) {
	keys := make([]string, 0, len(attrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeAttr(builder, key, attrs[key])
	}
}

func writeDataAttrs(builder *strings.Builder, data map[string]string) {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeAttr(builder, "data-"+key, data[key])
	}
}

func writeAttr(builder *strings.Builder, name string, value string) {
	if strings.TrimSpace(name) == "" || value == "" {
		return
	}
	builder.WriteByte(' ')
	builder.WriteString(attr(name))
	builder.WriteString(`="`)
	builder.WriteString(attr(value))
	builder.WriteByte('"')
}
