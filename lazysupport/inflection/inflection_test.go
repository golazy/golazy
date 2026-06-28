package inflection

import "testing"

var singularPluralPairs = map[string]string{
	"address":     "addresses",
	"agency":      "agencies",
	"alias":       "aliases",
	"analysis":    "analyses",
	"archive":     "archives",
	"axis":        "axes",
	"basis":       "bases",
	"box":         "boxes",
	"buffalo":     "buffaloes",
	"bus":         "buses",
	"case":        "cases",
	"category":    "categories",
	"child":       "children",
	"comment":     "comments",
	"crisis":      "crises",
	"database":    "databases",
	"datum":       "data",
	"day":         "days",
	"diagnosis":   "diagnoses",
	"edge":        "edges",
	"elf":         "elves",
	"equipment":   "equipment",
	"experience":  "experiences",
	"fish":        "fish",
	"fix":         "fixes",
	"half":        "halves",
	"horse":       "horses",
	"house":       "houses",
	"index":       "indices",
	"information": "information",
	"jeans":       "jeans",
	"louse":       "lice",
	"man":         "men",
	"matrix":      "matrices",
	"medium":      "media",
	"money":       "money",
	"mouse":       "mice",
	"movie":       "movies",
	"news":        "news",
	"octopus":     "octopi",
	"ox":          "oxen",
	"person":      "people",
	"photo":       "photos",
	"police":      "police",
	"portfolio":   "portfolios",
	"post":        "posts",
	"prize":       "prizes",
	"process":     "processes",
	"query":       "queries",
	"quiz":        "quizzes",
	"rice":        "rice",
	"safe":        "saves",
	"search":      "searches",
	"series":      "series",
	"sheep":       "sheep",
	"shoe":        "shoes",
	"species":     "species",
	"spokesman":   "spokesmen",
	"status":      "statuses",
	"switch":      "switches",
	"testis":      "testes",
	"tomato":      "tomatoes",
	"vertex":      "vertices",
	"virus":       "viri",
	"wife":        "wives",
	"woman":       "women",
}

func TestPluralize(t *testing.T) {
	for singular, plural := range singularPluralPairs {
		t.Run(singular, func(t *testing.T) {
			if got := Pluralize(singular); got != plural {
				t.Fatalf("Pluralize(%q) = %q, want %q", singular, got, plural)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	for singular, plural := range singularPluralPairs {
		t.Run(plural, func(t *testing.T) {
			if got := Singularize(plural); got != singular {
				t.Fatalf("Singularize(%q) = %q, want %q", plural, got, singular)
			}
		})
	}
}

func TestIrregularAddsCustomRules(t *testing.T) {
	inflector := newDefaultInflector()
	inflector.Irregular("console", "console")

	if got, want := inflector.Pluralize("console"), "console"; got != want {
		t.Fatalf("Pluralize(console) = %q, want %q", got, want)
	}
	if got, want := inflector.Singularize("console"), "console"; got != want {
		t.Fatalf("Singularize(console) = %q, want %q", got, want)
	}
}

func TestPluralizePreservesCamelCasePrefix(t *testing.T) {
	if got, want := Pluralize("CamelOctopus"), "CamelOctopi"; got != want {
		t.Fatalf("Pluralize(CamelOctopus) = %q, want %q", got, want)
	}
}

func TestSingularizePreservesCamelCasePrefix(t *testing.T) {
	if got, want := Singularize("CamelOctopi"), "CamelOctopus"; got != want {
		t.Fatalf("Singularize(CamelOctopi) = %q, want %q", got, want)
	}
}

func TestCamelize(t *testing.T) {
	tests := map[string]string{
		"http_connection_timeout": "HTTPConnectionTimeout",
		"multiple_http_calls":     "MultipleHTTPCalls",
		"my_account":              "MyAccount",
		"restful_controller":      "RESTfulController",
		"ssl_error":               "SSLError",
		"user-profile":            "UserProfile",
		"user profile":            "UserProfile",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := Camelize(input); got != want {
				t.Fatalf("Camelize(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestUnderscorize(t *testing.T) {
	tests := map[string]string{
		"HTTPConnectionTimeout": "http_connection_timeout",
		"MultipleHTTPCalls":     "multiple_http_calls",
		"MyAccount":             "my_account",
		"RESTfulController":     "restful_controller",
		"SSLError":              "ssl_error",
		"user-profile":          "user_profile",
		"user profile":          "user_profile",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := Underscorize(input); got != want {
				t.Fatalf("Underscorize(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestDasherize(t *testing.T) {
	if got, want := Dasherize("AdminPost"), "admin-post"; got != want {
		t.Fatalf("Dasherize(AdminPost) = %q, want %q", got, want)
	}
}

func TestTableize(t *testing.T) {
	tests := map[string]string{
		"fancyCategory":   "fancy_categories",
		"ham_and_egg":     "ham_and_eggs",
		"RawScaledScorer": "raw_scaled_scorers",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			if got := Tableize(input); got != want {
				t.Fatalf("Tableize(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestForeignKey(t *testing.T) {
	if got, want := ForeignKey("AdminPost"), "admin_post_id"; got != want {
		t.Fatalf("ForeignKey(AdminPost) = %q, want %q", got, want)
	}
}

func TestOrdinal(t *testing.T) {
	tests := map[int64]string{
		1:     "st",
		2:     "nd",
		3:     "rd",
		4:     "th",
		11:    "th",
		12:    "th",
		13:    "th",
		21:    "st",
		1002:  "nd",
		1003:  "rd",
		-1021: "st",
	}

	for input, want := range tests {
		t.Run(Ordinalize(input), func(t *testing.T) {
			if got := Ordinal(input); got != want {
				t.Fatalf("Ordinal(%d) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestOrdinalize(t *testing.T) {
	if got, want := Ordinalize(1003), "1003rd"; got != want {
		t.Fatalf("Ordinalize(1003) = %q, want %q", got, want)
	}
}
