package lazyaction

import "testing"

func TestValues(t *testing.T) {

	v := Values{
		"foo[bar]":      []string{"foo_bar"},
		"foo[bar][baz]": []string{"foo_bar_baz"},
		"asdf":          []string{"asdf"},
		"":              []string{"empty"},
	}

	a := v.Extract("foo")

	if a["bar"][0] != "foo_bar" {
		t.Fatal("foo[bar] not extracted")
	}
	if a["bar[baz]"][0] != "foo_bar_baz" {
		t.Fatal("foo[bar][baz] not extracted")
	}

}

type ValuesUser struct {
	Name string
	Age  int
}

func TestValuesLoad(t *testing.T) {

	v := Values{
		"user[Name]": []string{"Guillermo"},
		"user[age]":  []string{"23"},
	}

	u := ValuesUser{}

	err := v.Extract("user").Load(&u)
	if err != nil {
		t.Fatal(err)
	}

	if u.Name != "Guillermo" {
		t.Error("Name not loaded")
	}

	if u.Age != 23 {
		t.Error("Age not loaded")
	}

}
