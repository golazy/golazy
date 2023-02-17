package reflect

import (
	"strings"
	"testing"
)

type Base struct {
}

func (b Base) Render(input string) string {
	return "base" + input
}

type Base2 struct {
}

func (b *Base2) Render(input string) string {
	return "base2" + input
}

type Obj struct {
	Base
	Base2
}

func (o Obj) Render(input string) string {
	return "obj" + input
}	

func TestReflect_Inheritance(t *testing.T) {
	out := RecursiveCall("Render", "input").(string)
	t.Fatal(out)

}

type Stringer interface {
	String() string
}

func Rows(t []Method) string {
	out := []string{}
	for _, s := range t {
		out = append(out, s.String())
	}
	return strings.Join(out, "\n")
}

func TestReflect(t *testing.T) {
	data, err := ReflectAbout(Obj{})
	if err != nil {
		t.Fatal(err)
	}

	// "(Base::Base2) Render() string"

	t.Fatal(Rows(data))

}
