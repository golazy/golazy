package values

import "testing"

func TestValuesExtract(t *testing.T) {

	v := Values{
		"foo[bar]":            []string{"foo_bar"},
		"foo[bar][baz]":       []string{"foo_bar_baz"},
		"foo2[bar]":           []string{"trick"},
		"foo3[bar]":           []string{"trick"},
		"zap[trap][][name]":   []string{"good"},
		"zap[trap][33][name]": []string{"good"},
		"asdf":                []string{"asdf"},
		"":                    []string{"empty"},
	}

	a := v.Extract("foo")

	if a["bar"][0] != "foo_bar" {
		t.Fatal("foo[bar] not extracted")
	}
	if a["bar[baz]"][0] != "foo_bar_baz" {
		t.Fatal("foo[bar][baz] not extracted")
	}

	a = v.Extract("foo", "bar")
	if a["baz"][0] != "foo_bar_baz" {
		t.Fatal("foo[bar][baz] not extracted")
	}

}

func TestValueExtractListDestroy(t *testing.T) {
	v := Values{
		"person[friends][baz][name]":  []string{"baz"},
		"person[friends][][name]":     []string{"new_name"},
		"person[friends][][_destroy]": []string{"true"},
	}
	list := v.ExtractList("person", "friends")
	if len(list) != 1 {
		t.Error(list)
	}
}

func TestValuesExtractList(t *testing.T) {
	v := Values{
		"person[friends]":            []string{"foo_bar"},
		"person[friends][baz][name]": []string{"baz"},
		"person[friends][baz][age]":  []string{"33"},
		"person[friends][][name]":    []string{"new_name"},
		"person[friends][][age]":     []string{"new_age"},
		"asdf":                       []string{"asdf"},
		"":                           []string{"empty"},
	}

	list := v.ExtractList("person", "friends")

	test := func(key, property, value string) {
		if list[key][property][0] != value {
			t.Error("key", key, "property", property, "value", value)
		}
	}

	test("baz", "name", "baz")
	test("baz", "age", "33")
	test("", "name", "new_name")
	test("", "age", "new_age")

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
