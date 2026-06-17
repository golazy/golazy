package lazyforms

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"golazy.dev/lazyschema"
	"golazy.dev/lazyview"
)

var timeValueType = reflect.TypeOf(time.Time{})

func fieldHelper(inputType string) lazyview.Helper {
	return func(ctx *lazyview.Context, args ...any) (any, error) {
		form, fieldName, optionArgs, err := coerceFormAndField(ctx, args)
		if err != nil {
			return nil, err
		}
		options, err := collectFieldOptions(optionArgs...)
		if err != nil {
			return nil, err
		}
		if inputType != "" {
			options.inputType = inputType
		}
		field, err := form.Field(fieldName, options)
		if err != nil {
			return nil, err
		}
		return fragment(renderField(field)), nil
	}
}

func (f *Form) Field(fieldName string, options fieldOptions) (Field, error) {
	path, err := lazyschema.PathFor(f.modelTypeValue(), fieldName)
	if err != nil {
		return Field{}, err
	}
	name, err := lazyschema.FieldName(path)
	if err != nil {
		return Field{}, err
	}
	id, err := lazyschema.FieldID(path, lazyschema.WithPrefix(f.ModelKey))
	if err != nil {
		return Field{}, err
	}
	value := fieldValue(f.Model, fieldName)
	inputType := options.inputType
	if inputType == "" {
		inputType = defaultFieldType(value)
	}
	field := Field{
		Form:        f,
		GoPath:      fieldName,
		Path:        path,
		Name:        name,
		ID:          id,
		Label:       humanizeField(fieldName),
		Type:        inputType,
		Value:       value,
		Attrs:       copyStringMap(options.attrs),
		Placeholder: options.placeholder,
	}
	if options.id != "" {
		field.ID = options.id
	}
	if options.label != "" {
		field.Label = options.label
	}
	if options.class != "" {
		if field.Attrs == nil {
			field.Attrs = map[string]string{}
		}
		field.Attrs["class"] = options.class
	}
	for key, value := range options.data {
		if field.Attrs == nil {
			field.Attrs = map[string]string{}
		}
		field.Attrs["data-"+key] = value
	}
	if options.value != nil {
		field.Value = options.value
	}
	return field, nil
}

func (f *Form) modelTypeValue() any {
	if f.Model != nil {
		return f.Model
	}
	if f.ModelType == nil {
		return nil
	}
	return reflect.New(f.ModelType).Interface()
}

func helperFormFields(ctx *lazyview.Context, args ...any) (any, error) {
	form, err := activeForm(ctx)
	if err != nil {
		return nil, err
	}
	var builder strings.Builder
	for _, field := range form.Fields {
		builder.WriteString(renderField(field))
	}
	return fragment(builder.String()), nil
}

func helperSubmitButton(_ *lazyview.Context, args ...any) (any, error) {
	label := "Save"
	if len(args) > 0 {
		label = fmt.Sprint(args[0])
	}
	return fragment(`<button type="submit">` + esc(label) + `</button>`), nil
}

func helperFormValue(ctx *lazyview.Context, args ...any) (any, error) {
	form, fieldName, _, err := coerceFormAndField(ctx, args)
	if err != nil {
		return nil, err
	}
	return fieldValue(form.Model, fieldName), nil
}

func helperFormObject(ctx *lazyview.Context, _ ...any) (any, error) {
	form, err := activeForm(ctx)
	if err != nil {
		return nil, err
	}
	return form.Model, nil
}

func collectFields(form *Form, t reflect.Type, prefix string, indexes []string) []Field {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var fields []Field
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" || sf.Tag.Get("schema") == "-" || sf.Tag.Get("form") == "-" {
			continue
		}
		fieldPath := joinGoPath(prefix, sf.Name)
		ft := sf.Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft == timeValueType || ft.Kind() != reflect.Struct {
			field, err := form.Field(fieldPath, fieldOptions{})
			if err == nil {
				fields = append(fields, field)
			}
			continue
		}
		fields = append(fields, collectFields(form, ft, fieldPath, indexes)...)
	}
	return fields
}

func renderField(field Field) string {
	switch field.Type {
	case "textarea":
		return renderLabel(field, `<textarea`+fieldAttrs(field, false)+`>`+esc(formatValue(field.Value, field.Type))+`</textarea>`)
	case "checkbox":
		checked := ""
		if checkedValue(field.Value) {
			checked = ` checked`
		}
		return `<input type="hidden" name="` + attr(field.Name) + `" value="false">` +
			renderLabel(field, `<input type="checkbox"`+fieldAttrs(field, true)+` value="true"`+checked+`>`)
	case "hidden":
		return `<input type="hidden"` + fieldAttrs(field, true) + ` value="` + attr(formatValue(field.Value, field.Type)) + `">`
	case "file":
		return renderLabel(field, `<input type="file"`+fieldAttrs(field, true)+`>`)
	case "select":
		return renderLabel(field, `<select`+fieldAttrs(field, true)+`></select>`)
	default:
		return renderLabel(field, `<input type="`+attr(field.Type)+`"`+fieldAttrs(field, true)+` value="`+attr(formatValue(field.Value, field.Type))+`">`)
	}
}

func renderLabel(field Field, control string) string {
	if field.Label == "" {
		return control
	}
	return `<label for="` + attr(field.ID) + `">` + esc(field.Label) + ` ` + control + `</label>`
}

func fieldAttrs(field Field, includeID bool) string {
	var builder strings.Builder
	if includeID {
		writeAttr(&builder, "id", field.ID)
	}
	writeAttr(&builder, "name", field.Name)
	if field.Placeholder != "" {
		writeAttr(&builder, "placeholder", field.Placeholder)
	}
	writeAttrs(&builder, field.Attrs)
	return builder.String()
}

func defaultFieldType(value any) string {
	if value == nil {
		return "text"
	}
	v := reflect.ValueOf(value)
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "text"
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return "text"
	}
	if v.Type() == timeValueType {
		return "datetime-local"
	}
	switch v.Kind() {
	case reflect.Bool:
		return "checkbox"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	default:
		return "text"
	}
}

func fieldValue(model any, fieldPath string) any {
	if model == nil || isNil(model) {
		return nil
	}
	value := reflect.ValueOf(model)
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}
	for _, part := range strings.Split(fieldPath, ".") {
		if value.Kind() == reflect.Slice || value.Kind() == reflect.Array {
			index := 0
			if _, err := fmt.Sscan(part, &index); err != nil || index < 0 || index >= value.Len() {
				return nil
			}
			value = value.Index(index)
			continue
		}
		if value.Kind() != reflect.Struct {
			return nil
		}
		value = value.FieldByName(part)
		if !value.IsValid() {
			return nil
		}
		for value.Kind() == reflect.Ptr {
			if value.IsNil() {
				return nil
			}
			value = value.Elem()
		}
	}
	if value.IsValid() && value.CanInterface() {
		return value.Interface()
	}
	return nil
}

func formatValue(value any, inputType string) string {
	if value == nil {
		return ""
	}
	if t, ok := value.(time.Time); ok {
		return lazyschema.FormatTime(t, inputType)
	}
	return fmt.Sprint(value)
}

func checkedValue(value any) bool {
	if value == nil {
		return false
	}
	if checked, ok := value.(bool); ok {
		return checked
	}
	return fmt.Sprint(value) == "true"
}

func humanizeField(fieldPath string) string {
	parts := strings.Split(fieldPath, ".")
	name := parts[len(parts)-1]
	var builder strings.Builder
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			builder.WriteByte(' ')
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

func joinGoPath(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}
