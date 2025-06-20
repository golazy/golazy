package lazysupport

import (
	"testing"
)

var assert = struct {
	Equal func(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{})
}{
	Equal: func(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
		if expected != actual {
			t.Errorf("Expected %v, got %v", expected, actual)
		}
	},
}

var SingularToPlural = map[string]string{
	"search":      "searches",
	"switch":      "switches",
	"fix":         "fixes",
	"box":         "boxes",
	"process":     "processes",
	"address":     "addresses",
	"case":        "cases",
	"stack":       "stacks",
	"wish":        "wishes",
	"fish":        "fish",
	"jeans":       "jeans",
	"funky jeans": "funky jeans",
	"my money":    "my money",
	"category":    "categories",
	"query":       "queries",
	"ability":     "abilities",
	"agency":      "agencies",
	"movie":       "movies",
	"archive":     "archives",
	"index":       "indices",
	"wife":        "wives",
	"safe":        "saves",
	"half":        "halves",
	"move":        "moves",
	"salesperson": "salespeople",
	"person":      "people",
	"spokesman":   "spokesmen",
	"man":         "men",
	"woman":       "women",
	"basis":       "bases",
	"diagnosis":   "diagnoses",
	"diagnosis_a": "diagnosis_as",
	"datum":       "data",
	"medium":      "media",
	"stadium":     "stadia",
	"analysis":    "analyses",
	"my_analysis": "my_analyses",
	"node_child":  "node_children",
	"child":       "children",
	"experience":  "experiences",
	"day":         "days",
	"comment":     "comments",
	"foobar":      "foobars",
	"newsletter":  "newsletters",
	"old_news":    "old_news",
	"news":        "news",
	"series":      "series",
	"miniseries":  "miniseries",
	"species":     "species",
	"quiz":        "quizzes",
	"perspective": "perspectives",
	"ox":          "oxen",
	"photo":       "photos",
	"buffalo":     "buffaloes",
	"tomato":      "tomatoes",
	"dwarf":       "dwarves",
	"elf":         "elves",
	"information": "information",
	"equipment":   "equipment",
	"bus":         "buses",
	"status":      "statuses",
	"status_code": "status_codes",
	"mouse":       "mice",
	"louse":       "lice",
	"house":       "houses",
	"octopus":     "octopi",
	"virus":       "viri",
	"alias":       "aliases",
	"portfolio":   "portfolios",
	"vertex":      "vertices",
	"matrix":      "matrices",
	"matrix_fu":   "matrix_fus",
	"axis":        "axes",
	"taxi":        "taxis",
	"testis":      "testes",
	"crisis":      "crises",
	"rice":        "rice",
	"shoe":        "shoes",
	"horse":       "horses",
	"prize":       "prizes",
	"edge":        "edges",
	"database":    "databases",
	"brand":       "brands",
}

func TestPluralize(t *testing.T) {
	ClearCache()
	ShouldCache = false
	for singular, plural := range SingularToPlural {
		assert.Equal(t, plural, Pluralize(singular))
	}
}

func TestCachedPluralize(t *testing.T) {
	ClearCache()
	ShouldCache = true
	for singular, plural := range SingularToPlural {
		assert.Equal(t, plural, Pluralize(singular))
	}
}

func TestSingularize(t *testing.T) {
	ClearCache()
	ShouldCache = false
	for singular, plural := range SingularToPlural {
		assert.Equal(t, singular, Singularize(plural))
	}
}

func TestCachedSingularize(t *testing.T) {
	ClearCache()
	ShouldCache = true
	for singular, plural := range SingularToPlural {
		assert.Equal(t, singular, Singularize(plural))
	}
}

func TestMultipleAcronymInflections(t *testing.T) {
	term := "SSLError"
	assert.Equal(t, term, Camelize(Underscorize(term)))
}

func TestUnderscorize(t *testing.T) {
	term := "Lazy app"
	assert.Equal(t, "lazy_app", Underscorize(term))
}
