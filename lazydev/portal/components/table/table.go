package table

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

func cell(v reflect.Value) nodes.Element {
	var c any
	if !v.IsValid() {
		return html.Td("")
	}

	switch v.Kind() {
	case reflect.String:
		c = v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		c = fmt.Sprintf("%v", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		c = fmt.Sprintf("%v", v.Uint())
	case reflect.Float32, reflect.Float64:
		c = fmt.Sprintf("%v", v.Float())
	case reflect.Bool:
		c = fmt.Sprintf("%v", v.Bool())
	case reflect.Slice, reflect.Map, reflect.Interface:
		if v.IsNil() {
			return html.Td("")
		}
		fallthrough
	case reflect.Struct:

		data, err := json.MarshalIndent(v.Interface(), "", "  ")
		if err != nil {
			c = fmt.Sprintf("%v", err)
		} else {
			c = string(data)
		}
		c = html.Pre(c)
	default:
		c = fmt.Sprintf("%v", v)
	}

	return html.Td(c)
}

func New(data ...any) io.WriterTo {
	if len(data) == 0 {
		return bytes.NewBufferString("")
	}

	var structs reflect.Value
	var interfaces reflect.Value
	var elemT reflect.Type
	var cols []string
	for _, v := range data {
		t := reflect.TypeOf(v)
		if t.Kind() == reflect.Slice {
			sliceT := t.Elem()
			fmt.Println(sliceT.Kind())

			if sliceT.Kind() == reflect.Struct {
				structs = reflect.ValueOf(v)
				elemT = sliceT
				continue
			}

			if sliceT.Kind() == reflect.String {
				cols = v.([]string)
				continue
			}

			if sliceT.Kind() == reflect.Interface {
				interfaces = reflect.ValueOf(v)
				continue
			}
			panic("Expected to get an slice of struct or interfaces, or a slice of strings. Got" + fmt.Sprint(t, sliceT))
		}
		panic("Expected to get an slice of struct or interfaces, or a slice of strings. Got" + fmt.Sprint(reflect.TypeOf(v).Kind()))
	}
	if structs.IsValid() && interfaces.IsValid() {
		panic("Expected to get an slice of struct or interfaces. Got both")
	}

	// Lets start filling
	var head []any
	var rows []any
	var nCols int
	var nRows int

	if structs.IsValid() {
		nRows = structs.Len()

		// Find the columns if missing
		if cols == nil && elemT.Kind() == reflect.Struct {
			nCols = elemT.NumField()
			cols = make([]string, nCols)
			for i := 0; i < nCols; i++ {
				cols[i] = elemT.Field(i).Name
			}
		} else {
			nCols = len(cols)
		}

		// Fill the tbody
		for i := 0; i < nRows; i++ {
			row := make([]any, nCols)

			for j := 0; j < nCols; j++ {
				row[j] = cell(structs.Index(i).Field(j))
				//row[j] = html.Td(structs.Index(i).Field(j).String())
			}
			rows = append(rows, html.Tr(row...))
		}
	} else {
		if cols == nil {
			panic("When providing a slice of interfaces it is requried to provide a list of columns")
		}
		nCols = len(cols)

		// Fill the body
		nRows = interfaces.Len()

		for i := 0; i < nRows; i++ {
			row := make([]any, nCols)
			for j, c := range cols {
				a := interfaces.Index(i)
				elem := a.Elem()
				v := interfaces.Index(i).Elem().MethodByName(c)
				fmt.Println(a, elem, elem.Kind(), v.Type(), v.Type().String(), v.Type().NumIn())
				row[j] = html.Td()
				if !v.IsValid() {
					continue
				}
				if v.Type().NumIn() != 0 {
					continue
				}
				out := v.Call(nil)
				if len(out) < 1 {
					continue
				}
				row[j] = cell(out[0])

			}
			rows = append(rows, html.Tr(row...))
		}

	}

	// Fill the thead
	for _, v := range cols {
		head = append(head, html.Th(v))
	}

	return html.Table(
		html.Thead(html.Tr(head...)),
		html.Tbody(rows...),
	)

}
