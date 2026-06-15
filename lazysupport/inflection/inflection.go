package inflection

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Inflector struct {
	plurals        []rule
	singulars      []rule
	uncountables   []string
	acronyms       map[string]string
	acronymPattern *regexp.Regexp
}

type rule struct {
	pattern     *regexp.Regexp
	replacement string
}

var Default = newDefaultInflector()

var (
	upperWordsPattern = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
	lowerWordsPattern = regexp.MustCompile(`([a-z\d])([A-Z])`)
	separatorPattern  = regexp.MustCompile(`[_\-\s]+`)
)

func Pluralize(singular string) string {
	return Default.Pluralize(singular)
}

func Singularize(plural string) string {
	return Default.Singularize(plural)
}

func Camelize(term string) string {
	return Default.Camelize(term)
}

func Underscorize(term string) string {
	return Default.Underscorize(term)
}

func Dasherize(term string) string {
	return strings.ReplaceAll(Underscorize(term), "_", "-")
}

func Tableize(term string) string {
	return Pluralize(Underscorize(term))
}

func ForeignKey(term string) string {
	return Underscorize(term) + "_id"
}

func Ordinal(number int64) string {
	number = int64(math.Abs(float64(number)))
	if number%100 >= 11 && number%100 <= 13 {
		return "th"
	}
	switch number % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	default:
		return "th"
	}
}

func Ordinalize(number int64) string {
	return fmt.Sprintf("%d%s", number, Ordinal(number))
}

func (i *Inflector) Pluralize(singular string) string {
	return i.convert(singular, i.plurals)
}

func (i *Inflector) Singularize(plural string) string {
	return i.convert(plural, i.singulars)
}

func (i *Inflector) Camelize(term string) string {
	if strings.TrimSpace(term) == "" {
		return term
	}
	parts := separatorPattern.Split(term, -1)
	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if acronym, ok := i.acronyms[strings.ToLower(part)]; ok {
			out.WriteString(acronym)
			continue
		}
		out.WriteString(titleWord(part))
	}
	if out.Len() == 0 {
		return ""
	}
	return out.String()
}

func (i *Inflector) Underscorize(term string) string {
	if strings.TrimSpace(term) == "" {
		return term
	}
	if i.acronymPattern != nil {
		term = i.acronymPattern.ReplaceAllStringFunc(term, func(match string) string {
			return "_" + strings.ToLower(match)
		})
		term = strings.TrimPrefix(term, "_")
	}
	term = upperWordsPattern.ReplaceAllString(term, "${1}_${2}")
	term = lowerWordsPattern.ReplaceAllString(term, "${1}_${2}")
	term = separatorPattern.ReplaceAllString(term, "_")
	term = strings.Trim(term, "_")
	return strings.ToLower(term)
}

func (i *Inflector) convert(term string, rules []rule) string {
	if term == "" || i.isUncountable(term) {
		return term
	}
	for _, rule := range rules {
		if rule.pattern.MatchString(term) {
			return rule.pattern.ReplaceAllString(term, rule.replacement)
		}
	}
	return term
}

func (i *Inflector) isUncountable(term string) bool {
	term = strings.ToLower(term)
	for _, word := range i.uncountables {
		if strings.HasSuffix(term, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

func newDefaultInflector() *Inflector {
	inflector := &Inflector{
		acronyms: map[string]string{},
	}

	inflector.plural("", "s")
	inflector.plural("s", "s")
	inflector.plural("^(ax|test)is", "${1}es")
	inflector.plural("(octop|vir)us", "${1}i")
	inflector.plural("(octop|vir)i", "${1}i")
	inflector.plural("(alias|status)", "${1}es")
	inflector.plural("(bu)s", "${1}ses")
	inflector.plural("(buffal|tomat)o", "${1}oes")
	inflector.plural("([ti])um", "${1}a")
	inflector.plural("([ti])a", "${1}a")
	inflector.plural("sis", "ses")
	inflector.plural("(?:([^f])fe|([lr])f)", "${1}${2}ves")
	inflector.plural("(hive)", "${1}s")
	inflector.plural("([^aeiouy]|qu)y", "${1}ies")
	inflector.plural("(x|ch|ss|sh)", "${1}es")
	inflector.plural("(matr|vert|ind)(?:ix|ex)", "${1}ices")
	inflector.plural("^(m|l)ouse", "${1}ice")
	inflector.plural("^(m|l)ice", "${1}ice")
	inflector.plural("^(ox)", "${1}en")
	inflector.plural("^(oxen)", "${1}")
	inflector.plural("(quiz)", "${1}zes")

	inflector.singular("s", "")
	inflector.singular("(ss)", "${1}")
	inflector.singular("(n)ews", "${1}ews")
	inflector.singular("([ti])a", "${1}um")
	inflector.singular("((a)naly|(b)a|(d)iagno|(p)arenthe|(p)rogno|(s)ynop|(t)he)(sis|ses)", "${1}sis")
	inflector.singular("(^analy)(sis|ses)", "${1}sis")
	inflector.singular("([^f])ves", "${1}fe")
	inflector.singular("(hive)s", "${1}")
	inflector.singular("(tive)s", "${1}")
	inflector.singular("([lr])ves", "${1}f")
	inflector.singular("([^aeiouy]|qu)ies", "${1}y")
	inflector.singular("(s)eries", "${1}eries")
	inflector.singular("(m)ovies", "${1}ovie")
	inflector.singular("(x|ch|ss|sh)es", "${1}")
	inflector.singular("^(m|l)ice", "${1}ouse")
	inflector.singular("(bus)(es)?", "${1}")
	inflector.singular("(o)es", "${1}")
	inflector.singular("(shoe)s", "${1}")
	inflector.singular("(cris|test)(is|es)", "${1}is")
	inflector.singular("^(a)x[ie]s", "${1}xis")
	inflector.singular("(octop|vir)(us|i)", "${1}us")
	inflector.singular("(alias|status)(es)?", "${1}")
	inflector.singular("^(ox)en", "${1}")
	inflector.singular("(vert|ind)ices", "${1}ex")
	inflector.singular("(matr)ices", "${1}ix")
	inflector.singular("(quiz)zes", "${1}")
	inflector.singular("(database)s", "${1}")

	inflector.irregular("person", "people")
	inflector.irregular("man", "men")
	inflector.irregular("child", "children")
	inflector.irregular("sex", "sexes")
	inflector.irregular("move", "moves")
	inflector.irregular("zombie", "zombies")
	inflector.irregular("mombie", "mombies")

	inflector.uncountable(
		"equipment",
		"information",
		"rice",
		"money",
		"species",
		"series",
		"fish",
		"sheep",
		"jeans",
		"police",
	)

	inflector.acronym("HTTP")
	inflector.acronym("HTTPS")
	inflector.acronym("SSL")
	inflector.acronym("URL")
	inflector.acronym("URLs")
	inflector.acronym("API")
	inflector.acronym("APIs")
	inflector.acronym("REST")
	inflector.acronym("RESTful")
	inflector.acronym("USB")
	inflector.acronym("WWW")
	inflector.acronym("TCP")
	inflector.acronym("UDP")
	inflector.compileAcronyms()

	return inflector
}

func (i *Inflector) plural(pattern string, replacement string) {
	i.plurals = append([]rule{i.newRule(pattern, replacement)}, i.plurals...)
}

func (i *Inflector) singular(pattern string, replacement string) {
	i.singulars = append([]rule{i.newRule(pattern, replacement)}, i.singulars...)
}

func (i *Inflector) irregular(singular string, plural string) {
	quotedSingular := regexp.QuoteMeta(singular)
	quotedPlural := regexp.QuoteMeta(plural)
	i.plural(quotedSingular, plural)
	i.plural(quotedPlural, plural)
	i.singular(quotedSingular, singular)
	i.singular(quotedPlural, singular)
}

func (i *Inflector) uncountable(words ...string) {
	i.uncountables = append(i.uncountables, words...)
}

func (i *Inflector) acronym(word string) {
	i.acronyms[strings.ToLower(word)] = word
}

func (i *Inflector) compileAcronyms() {
	values := make([]string, 0, len(i.acronyms))
	for _, acronym := range i.acronyms {
		values = append(values, regexp.QuoteMeta(acronym))
	}
	sort.Slice(values, func(a int, b int) bool {
		return len(values[a]) > len(values[b])
	})
	if len(values) > 0 {
		i.acronymPattern = regexp.MustCompile(strings.Join(values, "|"))
	}
}

func (i *Inflector) newRule(pattern string, replacement string) rule {
	return rule{
		pattern:     regexp.MustCompile("(?i)" + pattern + "$"),
		replacement: replacement,
	}
}

func titleWord(word string) string {
	r, size := utf8.DecodeRuneInString(word)
	if r == utf8.RuneError && size == 0 {
		return word
	}
	return string(unicode.ToUpper(r)) + word[size:]
}
