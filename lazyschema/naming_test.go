package lazyschema

import (
	"testing"
	"time"
)

type phoneForm struct {
	Label  string
	Number string
}

type personForm struct {
	Name       string
	URLValue   string
	UserID     int
	Phone      phoneForm
	Phones     []phoneForm
	Birthday   time.Time
	CustomName string `schema:"custom_name"`
}

func TestFieldNameAndIDUseLowerCamelUnderscorePaths(t *testing.T) {
	tests := map[string]string{
		"Name":            "name",
		"URLValue":        "urlValue",
		"UserID":          "userID",
		"Phone.Label":     "phone_label",
		"Phones.0.Number": "phones_0_number",
		"CustomName":      "custom_name",
	}

	for field, want := range tests {
		t.Run(field, func(t *testing.T) {
			got, err := FieldNameFor(personForm{}, field)
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("FieldNameFor(%q) = %q, want %q", field, got, want)
			}
		})
	}

	id, err := FieldIDFor(personForm{}, "Phone.Label", WithPrefix("person"))
	if err != nil {
		t.Fatal(err)
	}
	if id != "person_phone_label" {
		t.Fatalf("FieldIDFor = %q, want person_phone_label", id)
	}
}

func TestDecoderUsesLowerCamelUnderscorePaths(t *testing.T) {
	var person personForm
	decoder := NewDecoder()
	err := decoder.Decode(&person, map[string][]string{
		"name":            {"Ada"},
		"urlValue":        {"https://example.test"},
		"userID":          {"42"},
		"phone_label":     {"mobile"},
		"phone_number":    {"555-0100"},
		"phones_0_label":  {"home"},
		"phones_0_number": {"555-0101"},
		"custom_name":     {"custom"},
		"birthday":        {"1985-12-10"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if person.Name != "Ada" || person.URLValue != "https://example.test" || person.UserID != 42 {
		t.Fatalf("decoded scalar fields = %#v", person)
	}
	if person.Phone.Label != "mobile" || person.Phone.Number != "555-0100" {
		t.Fatalf("decoded phone = %#v", person.Phone)
	}
	if len(person.Phones) != 1 || person.Phones[0].Label != "home" || person.Phones[0].Number != "555-0101" {
		t.Fatalf("decoded phones = %#v", person.Phones)
	}
	if person.CustomName != "custom" {
		t.Fatalf("CustomName = %q, want custom", person.CustomName)
	}
	if got := FormatTime(person.Birthday, "date"); got != "1985-12-10" {
		t.Fatalf("Birthday = %q, want 1985-12-10", got)
	}
}

func TestDecoderStillAcceptsDottedPaths(t *testing.T) {
	var person personForm
	err := NewDecoder().Decode(&person, map[string][]string{
		"Phone.Label":    {"mobile"},
		"Phones.0.Label": {"home"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if person.Phone.Label != "mobile" || person.Phones[0].Label != "home" {
		t.Fatalf("decoded person = %#v", person)
	}
}

func TestDecoderLimitsInput(t *testing.T) {
	var person personForm
	decoder := NewDecoder()
	decoder.MaxKeys(1)
	err := decoder.Decode(&person, map[string][]string{
		"name":   {"Ada"},
		"userID": {"42"},
	})
	if err == nil {
		t.Fatal("Decode succeeded, want max key error")
	}
}

type conflictForm struct {
	Name string
	Dupe string `schema:"name"`
}

func TestNameConflictReturnsError(t *testing.T) {
	var form conflictForm
	err := NewDecoder().Decode(&form, map[string][]string{"name": {"Ada"}})
	if err == nil {
		t.Fatal("Decode succeeded, want conflict error")
	}
}
