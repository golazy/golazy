package args

import "testing"

type TestStruct struct {
	s string
}

func (t *TestStruct) TestMethod(name string) string {
	return name
}

func (t *TestStruct) SetS(name string) {
	t.s = name
}

func TestFn2(t *testing.T) {

	NewFn2((&TestStruct{}).TestMethod)
	t.Error()

	/*
		instance := fn.NewInstance()

		outs := fn.InstanceCall(instance, InputSet{Values: map[string][]any{"string": {"hola"}}})

		if outs[0].String() != "hola" {
			t.Errorf("expected %s, got %s", "hola", outs[0].String())
		}
	*/
}
