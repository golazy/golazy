// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.gorilla file.

/*
Package lazyschema fills structs from submitted form values and derives the
field names used when rendering GoLazy forms.

The package is adapted from Gorilla schema. It keeps Gorilla's reflection-based
decoder model, converters, required/default tag options, and input-size
protection, but GoLazy owns the generated key format. That matters because the
same naming rules are used on both sides of a form: lazyforms calls this package
to render input names and ids, and lazycontroller.Base.Decode calls this package
to parse the submitted request form back into a struct.

The default field key is lower camel case per Go field segment, with underscores
between nested path segments:

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

The tag affects both decoding and name derivation. Dotted Gorilla-style keys
such as "Phone.Label" are still accepted by the decoder for compatibility, but
GoLazy-generated form names use underscore paths such as "phone_label".

The name helpers are useful outside the decoder when an application renders
forms without lazyforms:

	name, _ := lazyschema.FieldNameFor(Person{}, "Phone.Number")
	id, _ := lazyschema.FieldIDFor(Person{}, "Phone.Number", lazyschema.WithPrefix("person"))

FieldNameFor returns the value to place in the HTML name attribute. FieldIDFor
returns the matching DOM id; WithPrefix normally receives the model key, so a
Person.Phone.Number input can render as name="phone_number" and
id="person_phone_number". lazyforms uses ModelNameForType to derive that model
key, then uses PathFor, FieldName, and FieldID when rendering field helpers.

Use NewDecoder directly in standalone HTTP handlers, or use
lazycontroller.Base.Decode inside a GoLazy controller. Base.Decode parses the
request form, ignores unknown keys, treats empty submitted values as zero values,
and then delegates the actual struct decoding to this package. lazyroutes does
not consume lazyschema directly; its connection is through lazyforms, which uses
routes to choose form actions and lazyschema to choose field names.

The supported field types match Gorilla schema's core types: bools, strings,
integers, floats, structs, pointers to supported types, and slices. Custom types
can be registered with RegisterConverter. time.Time is registered by default for
common HTML date, time, and datetime input values.
*/
package lazyschema
