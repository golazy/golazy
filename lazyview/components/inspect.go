package components

import (
	"fmt"
	"io"
	"reflect"

	"golazy.dev/lazyview/components/table"
	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

func Inspect(s any) io.WriterTo {
	if s == nil {
		return html.Pre("nil")
	}

	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)

	switch t.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return html.Pre("nil")
		}
		return Inspect(v.Elem().Interface())
	case reflect.Slice:
		return table.New(s)
	case reflect.Struct:
		return inspectStruct(t, v)
	case reflect.Map:
		return inspectMap(s)
	}
	return html.Pre(fmt.Sprintf("%+v", s))

}

func inspectStruct(t reflect.Type, v reflect.Value) io.WriterTo {
	return html.Table(
		html.Thead(
			html.Tr(
				html.Th("Field"),
				html.Th("Value"),
			),
		),
		html.Tbody(
			nodes.Each(reflect.VisibleFields(t), func(f reflect.StructField) io.WriterTo {
				if f.IsExported() {
					return html.Tr(
						html.Td(f.Name),
						html.Td(fmt.Sprint(v.FieldByName(f.Name).Interface())),
					)
				}
				return nil
			}),
		),
	)
}

func inspectMap(s any) io.WriterTo {

	var content []any
	v := reflect.ValueOf(s)
	if v.Len() == 0 {
		return html.P("Empty map")
	}
	iter := v.MapRange()
	for iter.Next() {
		content = append(content, html.Tr(
			html.Td(Inspect(iter.Key().Interface())),
			html.Td(Inspect(iter.Value().Interface())),
		))
	}

	return html.Table(
		html.Caption("Inpsecting map"),
		html.Thead(
			html.Tr(
				html.Th("Field"),
				html.Th("Value"),
			),
		),
		html.Tbody(
			content...,
		),
	)
}
