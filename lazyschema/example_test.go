package lazyschema_test

import (
	"fmt"

	"golazy.dev/lazyschema"
)

func Example() {
	type Phone struct {
		Label  string
		Number string
	}
	type Person struct {
		Name  string
		Phone Phone
	}

	name, _ := lazyschema.FieldNameFor(Person{}, "Phone.Number")
	id, _ := lazyschema.FieldIDFor(Person{}, "Phone.Number", lazyschema.WithPrefix("person"))
	fmt.Println(name, id)

	var person Person
	decoder := lazyschema.NewDecoder()
	err := decoder.Decode(&person, map[string][]string{
		"name":         {"Ada"},
		"phone_label":  {"mobile"},
		"phone_number": {"555-0100"},
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s: %s %s\n", person.Name, person.Phone.Label, person.Phone.Number)

	// Output:
	// phone_number person_phone_number
	// Ada: mobile 555-0100
}
