// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.gorilla file.

/*
Package lazyschema fills structs from form values and generates the field names
used by GoLazy forms.

The package is adapted from Gorilla schema. It keeps Gorilla's reflection-based
decoder model, converters, required/default tag options, and slice protection,
but GoLazy owns the generated key format.

The default field key is lower-camel per Go field segment, with underscores for
nested paths:

	type Person struct {
		Name  string
		Phone Phone
	}

	type Phone struct {
		Label  string
		Number string
	}

	values := map[string][]string{
		"name":         {"Ada"},
		"phone_label":  {"mobile"},
		"phone_number": {"555-0100"},
	}

	var person Person
	decoder := lazyschema.NewDecoder()
	err := decoder.Decode(&person, values)

Slices of structs include explicit numeric indexes:

	type Person struct {
		Phones []Phone
	}

	values := map[string][]string{
		"phones_0_label":  {"home"},
		"phones_0_number": {"555-0100"},
		"phones_1_label":  {"work"},
		"phones_1_number": {"555-0101"},
	}

Use the "schema" struct tag to override names or ignore fields:

	type Person struct {
		Name  string `schema:"fullName"`
		Admin bool   `schema:"-"`
	}

The same package generates field names and ids, so form helpers and decoding
stay aligned:

	name, _ := lazyschema.FieldNameFor(Person{}, "Phone.Number")
	id, _ := lazyschema.FieldIDFor(Person{}, "Phone.Number", lazyschema.WithPrefix("person"))

The supported field types match Gorilla schema's core types: bools, strings,
integers, floats, structs, pointers to supported types, and slices. Custom types
can be registered with RegisterConverter. time.Time is registered by default for
common HTML date, time, and datetime input values.
*/
package lazyschema
