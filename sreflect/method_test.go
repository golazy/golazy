package sreflect

import (
	"fmt"
	"reflect"
	"testing"
)

type CallBase struct {
	Value string
	CallBase2
	*CallBase3
}
type CallBase2 struct {
	Value string
	*CallBase3
}

type CallBase3 struct {
	Value string
}

func (c CallBase) Method(value int) string {
	return fmt.Sprintf("%s %d", c.Value, value)
}

func (c *CallBase2) Method(value string) string {
	return fmt.Sprintf("%s %s", c.Value, value)
}
func (c *CallBase3) Method(value error) string {
	return fmt.Sprintf("%s%s", value, c.Value)
}

func find(methods []*Method, name string) *Method {
	for _, m := range methods {
		println(m.String())
		if m.String() == name {
			return m
		}
	}
	return nil
}

func TestMethodCall(t *testing.T) {

	s := &CallBase{
		Value: "hello",
		CallBase2: CallBase2{
			Value: "world",
			CallBase3: &CallBase3{
				Value: "!",
			},
		},
	}
	v := reflect.ValueOf(s)

	info := Reflect(s)

	resolver := func(t reflect.Type) (reflect.Value, error) {
		switch t.Kind() {
		case reflect.String:
			return reflect.ValueOf("potatos"), nil
		case reflect.Int:
			return reflect.ValueOf(42), nil
		default:
			errorType := reflect.TypeOf((*error)(nil)).Elem()
			if t.Implements(errorType) {
				return reflect.ValueOf(fmt.Errorf("noooo")), nil
			}
			return reflect.ValueOf(nil), fmt.Errorf("type %q not found", t)
		}
	}

	// Call a nested method
	t.Run("nested", func(t *testing.T) {
		m := find(info.AllMethods(), "*sreflect.CallBase.CallBase2.Method")
		if m == nil {
			t.Fatal("Expected to find method *sreflect.CallBase.Method")
		}

		res, err := m.Call(v, resolver)
		if err != nil {
			t.Fatal(err)
		}
		result := res[0].String()
		if result != "world potatos" {
			t.Fatal("Expected hello 42 got", result)
		}
	})
	t.Run("nested pointer", func(t *testing.T) {
		m := find(info.AllMethods(), "*sreflect.CallBase.CallBase2.CallBase3.Method")
		if m == nil {
			t.Fatal("Expected to find method *sreflect.CallBase.CallBase2.CallBase3.Method")
		}

		res, err := m.Call(v, resolver)
		if err != nil {
			t.Fatal(err)
		}
		result := res[0].String()
		if result != "noooo!" {
			t.Fatal("Expected hello 42 got", result)
		}

	})

	// Call a top level method
	t.Run("top", func(t *testing.T) {
		m := find(info.AllMethods(), "*sreflect.CallBase.Method")
		if m == nil {
			names := []string{}
			for _, m := range info.AllMethods() {
				names = append(names, m.String())
			}
			t.Fatalf("Expected to find method *sreflect.CallBase.Method got %v", names)
		}

		res, err := m.Call(v, resolver)
		if err != nil {
			t.Fatal(err)
		}
		result := res[0].String()
		if result != "hello 42" {
			t.Fatal("Expected hello 42 got", result)
		}
	})

}
