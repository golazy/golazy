package lazyforms

import (
	"fmt"
	"html/template"
	"maps"
	"reflect"
	"strings"

	"golazy.dev/lazyview"
)

const htmlContentType = "text/html; charset=utf-8"

func Helpers(router Router) map[string]any {
	return map[string]any{
		"form_for":          helperFormFor(router),
		"delete_button_for": helperDeleteButtonFor(router),
		"form_fields":       lazyview.Helper(helperFormFields),
		"form_field":        fieldHelper(""),
		"text_field":        fieldHelper("text"),
		"email_field":       fieldHelper("email"),
		"password_field":    fieldHelper("password"),
		"number_field":      fieldHelper("number"),
		"date_field":        fieldHelper("date"),
		"time_field":        fieldHelper("time"),
		"datetime_field":    fieldHelper("datetime-local"),
		"month_field":       fieldHelper("month"),
		"week_field":        fieldHelper("week"),
		"checkbox_field":    fieldHelper("checkbox"),
		"textarea":          fieldHelper("textarea"),
		"select_field":      fieldHelper("select"),
		"hidden_field":      fieldHelper("hidden"),
		"file_field":        fieldHelper("file"),
		"submit_button":     lazyview.Helper(helperSubmitButton),
		"form_value":        lazyview.Helper(helperFormValue),
		"form_object":       lazyview.Helper(helperFormObject),
		"form_id_value":     helperFormIDValue,
		"form_class_value":  helperFormClassValue,
		"form_action":       FormAction,
		"form_route":        FormRoute,
		"form_method":       FormMethod,
		"form_id":           FormID,
		"form_class":        FormClass,
		"form_add_class":    FormAddClass,
		"form_file":         FormFile,
		"form_model":        FormModel,
		"form_scope":        FormScope,
		"form_multipart":    func() Option { return FormMultipart() },
		"form_attr":         FormAttr,
		"form_data":         FormData,
		"field_label":       FieldLabel,
		"field_type":        FieldType,
		"field_id":          FieldID,
		"field_class":       FieldClass,
		"field_value":       FieldValue,
		"field_placeholder": FieldPlaceholder,
		"field_attr":        FieldAttr,
		"field_data":        FieldData,
	}
}

func helperFormFor(router Router) lazyview.Helper {
	return func(ctx *lazyview.Context, args ...any) (any, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("lazyforms: form_for requires model and context")
		}
		model := args[0]
		originalDot := args[1]
		options, err := collectOptions(args[2:]...)
		if err != nil {
			return nil, err
		}

		form, err := buildForm(router, model, originalDot, options)
		if err != nil {
			return nil, err
		}
		body, err := ctx.Views.RenderString(lazyview.Options{
			Context:    ctx.Context,
			Request:    ctx.Request,
			Variables:  formVariables(originalDot, form),
			Data:       formVariables(originalDot, form),
			Route:      ctx.Route,
			Namespace:  ctx.Namespace,
			Controller: ctx.Controller,
			Partial:    form.Partial,
			Format:     ctx.Format,
			UseLayout:  false,
		})
		if err != nil {
			return nil, err
		}
		return fragment(renderForm(form, body)), nil
	}
}

func helperDeleteButtonFor(router Router) lazyview.Helper {
	return func(_ *lazyview.Context, args ...any) (any, error) {
		if len(args) == 0 {
			return nil, fmt.Errorf("lazyforms: delete_button_for requires model")
		}
		label := "Delete"
		optionArgs := make([]any, 0, len(args)-1)
		for _, arg := range args[1:] {
			if arg == nil {
				continue
			}
			if _, ok := arg.(Option); ok {
				optionArgs = append(optionArgs, arg)
				continue
			}
			if value, ok := arg.(string); ok && label == "Delete" {
				label = value
				continue
			}
			return nil, fmt.Errorf("lazyforms: delete_button_for argument %T is not supported", arg)
		}
		options, err := collectOptions(optionArgs...)
		if err != nil {
			return nil, err
		}
		model := args[0]
		if options.action == "" {
			if options.routeName != "" {
				if routeRouter, ok := router.(routeRouter); ok {
					action, err := routeRouter.PathFor(options.routeName, options.routeValues...)
					if err != nil {
						return nil, err
					}
					options.action = action
				} else {
					return nil, fmt.Errorf("lazyforms: form_route requires a router with PathFor")
				}
			}
		}
		if options.action == "" {
			if router == nil {
				return nil, fmt.Errorf("lazyforms: router is required for delete_button_for")
			}
			action, err := router.PathForModel(model, "delete")
			if err != nil {
				return nil, err
			}
			options.action = action
		}
		html := `<form action="` + attr(options.action) + `" method="post">` +
			`<input type="hidden" name="_method" value="delete">` +
			`<button type="submit">` + esc(label) + `</button></form>`
		return fragment(html), nil
	}
}

func collectOptions(args ...any) (formOptions, error) {
	var options formOptions
	for _, arg := range args {
		if arg == nil {
			continue
		}
		option, ok := arg.(Option)
		if !ok {
			return options, fmt.Errorf("lazyforms: option %T is not a form option", arg)
		}
		option(&options)
	}
	return options, nil
}

func formVariables(original any, form *Form) map[string]any {
	variables := map[string]any{}
	if values, ok := original.(map[string]any); ok {
		maps.Copy(variables, values)
	} else if original != nil {
		variables["Context"] = original
	}
	variables["Form"] = form
	variables["Model"] = form.Model
	variables["FormObject"] = form.Model
	variables["FormData"] = form
	variables["FormOptions"] = form
	variables["FormFields"] = form.Fields
	return variables
}

func fragment(body string) lazyview.Fragment {
	return lazyview.Fragment{Body: body, ContentType: htmlContentType}
}

func helperFormIDValue(model any) (string, error) {
	form, err := modelFormDefaults(nil, model, nil)
	if err != nil {
		return "", err
	}
	return form.ID, nil
}

func helperFormClassValue(model any) (string, error) {
	form, err := modelFormDefaults(nil, model, nil)
	if err != nil {
		return "", err
	}
	return form.Class, nil
}

func activeForm(ctx *lazyview.Context) (*Form, error) {
	if values, ok := ctx.Data.(map[string]any); ok {
		if form, ok := values["Form"].(*Form); ok {
			return form, nil
		}
	}
	return nil, fmt.Errorf("lazyforms: no active form")
}

func coerceFormAndField(ctx *lazyview.Context, args []any) (*Form, string, []any, error) {
	if len(args) == 0 {
		return nil, "", nil, fmt.Errorf("lazyforms: field helper requires a field name")
	}
	if form, ok := args[0].(*Form); ok {
		if len(args) < 2 {
			return nil, "", nil, fmt.Errorf("lazyforms: explicit form helper requires a field name")
		}
		field, ok := args[1].(string)
		if !ok {
			return nil, "", nil, fmt.Errorf("lazyforms: field name must be a string")
		}
		return form, field, args[2:], nil
	}
	field, ok := args[0].(string)
	if !ok {
		return nil, "", nil, fmt.Errorf("lazyforms: field name must be a string")
	}
	form, err := activeForm(ctx)
	return form, field, args[1:], err
}

func collectFieldOptions(args ...any) (fieldOptions, error) {
	var options fieldOptions
	for _, arg := range args {
		if arg == nil {
			continue
		}
		option, ok := arg.(FieldOption)
		if !ok {
			return options, fmt.Errorf("lazyforms: option %T is not a field option", arg)
		}
		option(&options)
	}
	return options, nil
}

func attr(value string) string {
	return template.HTMLEscapeString(value)
}

func esc(value string) string {
	return template.HTMLEscapeString(value)
}

func isNil(model any) bool {
	if model == nil {
		return true
	}
	value := reflect.ValueOf(model)
	return value.Kind() == reflect.Pointer && value.IsNil()
}

func methodFor(model any, explicit string) (string, string) {
	if explicit != "" {
		method := strings.ToLower(explicit)
		if method == "get" || method == "post" {
			return method, ""
		}
		return "post", method
	}
	if !isNil(model) {
		if resource, ok := model.(Resource); ok && resource.Persisted() {
			return "post", "patch"
		}
	}
	return "post", ""
}
