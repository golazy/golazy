package sreflect

import (
	"io"
	"reflect"
	"slices"
	"testing"
)

type BaseStruct struct {
	Name string
	NestedStruct
	*NestedStruct2
}

func (b *BaseStruct) Init(name string) {
	b.Name = name
}

type NestedStruct2 struct {
	Name string
	NestedStruct3
}

type NestedStruct3 struct {
	Name string
}

func (n NestedStruct3) Init(name string) {
	n.Name = name
}

type NestedStruct struct {
	Name string
}

func (n *NestedStruct) Init(name string) {
	n.Name = name
}

func TestSReflect(t *testing.T) {

	a := &BaseStruct{}

	info := Reflect(a)

	if info.Name() != "*sreflect.BaseStruct" {
		t.Errorf("Expected BaseStruct got %s", info.Name())
	}
}

func TestReflectOnNonStruct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when passing non-struct type")
		}
	}()

	Reflect(123) // Passing a non-struct type should cause a panic
}

func TestReflectOnNilPointer(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when passing nil pointer")
		}
	}()

	var a *BaseStruct
	Reflect(a) // Passing a nil pointer should cause a panic
}

func TestStructInfo_Struct(t *testing.T) {
	a := &BaseStruct{
		NestedStruct2: &NestedStruct2{},
	}
	si := Reflect(a)
	structs := si.AllStructs()

	names := []string{}
	for _, s := range structs {
		names = append(names, s.String())
	}
	if !slices.Equal(names, []string{
		"*sreflect.BaseStruct",
		"*sreflect.BaseStruct.NestedStruct",
		"*sreflect.BaseStruct.NestedStruct2",
		"*sreflect.BaseStruct.NestedStruct2.NestedStruct3",
	}) {
		t.Errorf("Unexpected structs: %+v", names)
	}
}

type StructA struct {
	*StructA
	*StructB
	*StructC
	Other *StructC
	io.Writer
}
type StructB struct {
	*StructA
	*StructC
}
type StructC struct {
	StructA
	StructB
}

func TestRecursiveStructs(t *testing.T) {
	e := &StructA{StructB: &StructB{}}
	e.StructB.StructA = e

	s := Reflect(e)
	structs := s.AllStructs()
	names := []string{}
	for _, s := range structs {
		names = append(names, s.String())
	}
	if !slices.Equal(names, []string{
		"*sreflect.StructA",
		"*sreflect.StructA.StructB",
		"*sreflect.StructA.StructB.StructC",
		"*sreflect.StructA.StructC",
		"*sreflect.StructA.StructC.StructB",
	}) {
		t.Errorf("Unexpected structs: %+v", names)
	}
}

func TestStructInfo_Methods(t *testing.T) {
	a := &BaseStruct{}
	si := Reflect(a)
	t.Run("direct methods", func(t *testing.T) {
		methods := si.Methods

		if len(methods) != 1 {
			t.Fatalf("Expected 1 method got %d", len(methods))
		}
		if methods[0].Name != "Init" {
			t.Errorf("Expected Init got %s", methods[0].Name)
		}
	})

	t.Run("nested methods", func(t *testing.T) {
		allMethods := si.AllMethods()
		methodNames := []string{}
		for _, m := range allMethods {
			methodNames = append(methodNames, m.String())
		}
		if !reflect.DeepEqual(methodNames, []string{
			"*sreflect.BaseStruct.Init",
			"*sreflect.BaseStruct.NestedStruct.Init",
			"*sreflect.BaseStruct.NestedStruct2.Init",
			"*sreflect.BaseStruct.NestedStruct2.NestedStruct3.Init",
		}) {
			t.Errorf("Expected Init, Init, Init got %v", methodNames)

		}
		if len(allMethods) != 4 {
			t.Fatalf("Expected AllMethods to return 3 methods got %d", len(allMethods))
		}
	})

}
