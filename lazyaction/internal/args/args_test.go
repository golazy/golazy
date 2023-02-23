package args

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type myData struct{}

type controller struct {
}

func (c controller) Index() string {
	return "index"
}

func (c *controller) Show(id int, d *myData) (int, error) {
	return id, nil
}

func TestExtractArgs(t *testing.T) {

	expect := func(action any, ins, out []string, err error) {

		val := reflect.TypeOf(action)

		args, returns, e := ExtractArgs(val)
		if e != err {
			t.Errorf("expected error %v, got %v", err, e)
		}
		if !reflect.DeepEqual(args, ins) {
			t.Errorf("expected args %v, got %v", ins, args)
		}
		if !reflect.DeepEqual(returns, out) {
			t.Errorf("expected returns %v, got %v", out, returns)
		}
	}

	expect(nil, nil, nil, ErrNilAction)
	expect(1, nil, nil, ErrNonFuncAction)
	expect(func() {}, []string{}, []string{}, nil)

	c := controller{}
	expect(c.Index, []string{}, []string{"string"}, nil)
	expect(c.Show, []string{"int", "*args.myData"}, []string{"int", "error"}, nil)

	cPtr := &controller{}
	expect(cPtr.Index, []string{}, []string{"string"}, nil)
	expect(cPtr.Show, []string{"int", "*args.myData"}, []string{"int", "error"}, nil)
}

func TestFn(t *testing.T) {
	sampleFn := func(name string, age int) (string, error) {
		return fmt.Sprintf("Hola %s, tienes %d años", name, age), nil
	}
	fn := NewFn(sampleFn)

	// Check errors
	_, err := fn.Call(InputSet{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Prepare InputSet
	stringGenerator := func(age int) (string, error) {
		return fmt.Sprintf("Juan(%d)", age), nil
	}
	stringGenFn := NewGen(stringGenerator)

	is := InputSet{
		Generators: map[string][]Gen{"string": {stringGenFn}},
		Values:     map[string][]any{"int": {10}},
	}

	outs, err := fn.Call(is)
	if err != nil {
		t.Fatal(err)
	}

	s := outs[0].String()

	if s != "Hola Juan(10), tienes 10 años" {
		t.Errorf("expected %s, got %s", "", s)
	}

	if !outs[1].IsNil() {
		t.Error("expected nil, got error")
	}

}

func TestFn_SameArg(t *testing.T) {
	sampleFn := func(id, age int) (string, error) {
		return fmt.Sprintf("%d=>%d", id, age), nil
	}
	fn := NewFn(sampleFn)

	_, err := fn.Call(InputSet{})
	if errors.Is(err, new(ErrArgumentNotFound)) {
		t.Fatal(err)
	}

	is := InputSet{
		Values: map[string][]any{"int": {10, 7}},
	}

	outs, err := fn.Call(is)
	if err != nil {
		t.Fatal(err)
	}
	s := outs[0].String()

	if s != "10=>7" {
		t.Errorf("expected %s, got %s", "", s)
	}

	if !outs[1].IsNil() {
		t.Error("expected nil, got error")
	}

}
