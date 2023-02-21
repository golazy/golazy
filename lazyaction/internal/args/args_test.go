package args

import (
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
		args, returns, e := ExtractArgs(action)
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
