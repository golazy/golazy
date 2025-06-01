package lazysupport

import (
	"testing"
)

type DomIDTestWithField struct {
	ID uint
}

type DomIDTestWithMethod struct {
}

func (d DomIDTestWithMethod) ID() uint {
	return 1
}

type DomIDTestWithToParam struct {
}

func (d DomIDTestWithToParam) ToParam() string {
	return "88"
}

type DomIDTestWithInterface struct {
}

func (d DomIDTestWithInterface) DomID() string {
	return "super_99"
}

type DomIDTestNotCompatible struct {
}

func TestDomID(t *testing.T) {

	expect := func(obj any, expected string, expectedError string) {
		t.Helper()
		result, err := DomID(obj)
		if err != nil {
			if err.Error() != expectedError {
				t.Errorf("Expected %T%+v to produce error %q, got %q", obj, obj, expectedError, err.Error())
			}

		} else {
			if result != expected {
				t.Errorf("Expected %T%+v to be %q, got %q", obj, obj, expected, result)
			}
		}
	}

	expect(DomIDTestWithField{ID: 1}, "dom_id_test_with_field_1", "")
	expect(&DomIDTestWithField{ID: 1}, "dom_id_test_with_field_1", "")
	expect(DomIDTestWithMethod{}, "dom_id_test_with_method_1", "")
	expect(&DomIDTestWithMethod{}, "dom_id_test_with_method_1", "")
	expect(DomIDTestWithInterface{}, "super_99", "")
	expect(&DomIDTestWithInterface{}, "super_99", "")
	expect(DomIDTestNotCompatible{}, "", "cannot get ID from lazysupport.DomIDTestNotCompatible")
	expect(&DomIDTestNotCompatible{}, "", "cannot get ID from *lazysupport.DomIDTestNotCompatible")
	expect(DomIDTestWithToParam{}, "dom_id_test_with_to_param_88", "")
	expect(nil, "", "cannot get ID from nil")
	expect(4, "", "cannot get ID from int")
	expect("asdf", "", "cannot get ID from string")

}
