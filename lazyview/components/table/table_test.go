package table

import (
	"bytes"
	"testing"

	"golazy.dev/lazyview/nodes"
)

func TestTable(t *testing.T) {

	table := New([]struct{ Name, Age string }{{"John", "20"}})

	nodes.Beautify = false
	b := &bytes.Buffer{}
	table.WriteTo(b)

	expectation := `<table><thead><tr><th>Name<th>Age<tbody><tr><td>John<td>20</table>`
	if b.String() != expectation {
		t.Errorf("Expected %s, got %s", expectation, b.String())
	}

}

func TestTableColumns(t *testing.T) {

	table := New([]string{"Name"}, []struct{ Name, Age string }{{"John", "20"}})

	nodes.Beautify = false
	b := &bytes.Buffer{}
	table.WriteTo(b)

	expectation := `<table><thead><tr><th>Name<tbody><tr><td>John</table>`
	if b.String() != expectation {
		t.Errorf("Expected %s, got %s", expectation, b.String())
	}
}

type col string

func (c col) Alive() bool {
	return true
}

func (c col) Name() string {
	return string(c)
}

func TestTableInterface(t *testing.T) {

	table := New(
		[]string{"Name"},
		[]any{col("Pepe")},
	)

	nodes.Beautify = false
	b := &bytes.Buffer{}
	table.WriteTo(b)

	expectation := `<table><thead><tr><th>Name<tbody><tr><td>Pepe</table>`
	if b.String() != expectation {
		t.Errorf("Expected %s, got %s", expectation, b.String())
	}

}
