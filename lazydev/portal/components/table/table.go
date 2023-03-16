package table

import (
	"fmt"
	"io"
	"reflect"

	"golazy.dev/lazyview/html"
)

func New(data any) io.WriterTo {
	if reflect.TypeOf(data).Kind() != reflect.Slice {
		panic("Expected to get an slice. Got" + fmt.Sprint(reflect.TypeOf(data).Kind()))
	}

	t := reflect.TypeOf(data).Elem()

	if t.Kind() != reflect.Struct {
		panic("Expected to get an slice of struct. Got" + fmt.Sprint(t.Kind()))
	}

	nCols := t.NumField()
	cols := make([]string, nCols)
	head := make([]any, nCols)
	for i := 0; i < nCols; i++ {
		cols[i] = t.Field(i).Name
		head[i] = html.Th(cols[i])
	}

	v := reflect.ValueOf(data)
	nRows := v.Len()

	rows := make([]any, nRows)

	for i := 0; i < nRows; i++ {
		row := make([]any, nCols)
		for j := 0; j < nCols; j++ {
			row[j] = html.Td(v.Index(i).Field(j).String())
		}
		rows[i] = html.Tr(row...)
	}

	return html.Table(
		html.Thead(html.Tr(head...)),
		html.Tbody(rows...),
	)

}
