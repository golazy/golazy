package lazysupport

import (
	"fmt"
	"testing"
)

type NameOfTest struct {
}

func TestNameOf(t *testing.T) {

	expect := func(obj any, toBe string, shouldPanic bool) {
		t.Helper()
		defer func() {
			err := recover()
			if err == nil {
				if shouldPanic {
					t.Errorf("Expecting %t panic", obj)
				}
				return

			}

			if !shouldPanic {
				t.Errorf("Expecteing %t to panic", obj)
				return
			}

			errMsg := ""

			if panicErr, ok := err.(error); ok {
				errMsg = panicErr.Error()
			} else {
				errMsg = fmt.Sprint(err)
			}
			if errMsg != toBe {
				t.Errorf("Expecting %t to be %q Got %q", obj, toBe, errMsg)
			}
		}()
		name := NameOf(obj)
		if name != toBe {
			t.Errorf("Expecting %t Got %q", obj, toBe)
		}
	}

	expect(nil, "NameOf called with nil", true)
	expect(1, "NameOf called with non struct, slice or map", true)
	expect("asdf", "NameOf called with non struct, slice or map", true)
	var a *NameOfTest = nil
	expect(a, "NameOfTest", false)
	expect(NameOfTest{}, "NameOfTest", false)
	expect(&NameOfTest{}, "NameOfTest", false)
	expect([]NameOfTest{}, "NameOfTests", false)
	expect([]*NameOfTest{}, "NameOfTests", false)
	expect([]**NameOfTest{}, "NameOfTests", false)
	expect([]***NameOfTest{}, "NameOfTests", false)
	expect([]****NameOfTest{}, "NameOfTests", false)
	expect(map[any]NameOfTest{}, "NameOfTests", false)
	expect(map[any]*NameOfTest{}, "NameOfTests", false)
	expect(map[any]**NameOfTest{}, "NameOfTests", false)
	expect(map[any]***NameOfTest{}, "NameOfTests", false)
	expect([3]NameOfTest{}, "NameOfTests", false)
	expect([3]*NameOfTest{}, "NameOfTests", false)
	expect([3]**NameOfTest{}, "NameOfTests", false)

}

func TestNameOfWithPackage(t *testing.T) {

	expect := func(obj any, toBe string, shouldPanic bool) {
		t.Helper()
		defer func() {
			err := recover()
			if err == nil {
				if shouldPanic {
					t.Errorf("Expecting %t panic", obj)
				}
				return

			}

			if !shouldPanic {
				t.Errorf("Expecteing %t to panic", obj)
				return
			}

			errMsg := ""

			if panicErr, ok := err.(error); ok {
				errMsg = panicErr.Error()
			} else {
				errMsg = fmt.Sprint(err)
			}
			if errMsg != toBe {
				t.Errorf("Expecting %t to be %q Got %q", obj, toBe, errMsg)
			}
		}()
		name := NameOfWithPackage(obj)
		if name != toBe {
			t.Errorf("Expecting %t to be %q Got %q", obj, toBe, name)
		}
	}

	expect(nil, "NameOf called with nil", true)
	expect(1, "NameOf called with non struct, slice or map", true)
	expect("asdf", "NameOf called with non struct, slice or map", true)
	var a *NameOfTest = nil
	expect(a, "lazysupport.NameOfTest", false)
	expect(NameOfTest{}, "lazysupport.NameOfTest", false)
	expect(&NameOfTest{}, "lazysupport.NameOfTest", false)
	expect([]NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([]*NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([]**NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([]***NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([]****NameOfTest{}, "lazysupport.NameOfTests", false)
	expect(map[any]NameOfTest{}, "lazysupport.NameOfTests", false)
	expect(map[any]*NameOfTest{}, "lazysupport.NameOfTests", false)
	expect(map[any]**NameOfTest{}, "lazysupport.NameOfTests", false)
	expect(map[any]***NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([3]NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([3]*NameOfTest{}, "lazysupport.NameOfTests", false)
	expect([3]**NameOfTest{}, "lazysupport.NameOfTests", false)

}
