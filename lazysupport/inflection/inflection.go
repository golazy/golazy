package inflection

import "strings"

func Pluralize(singular string) string {
	if strings.HasSuffix(singular, "y") {
		return strings.TrimSuffix(singular, "y") + "ies"
	}
	if strings.HasSuffix(singular, "s") {
		return singular
	}
	return singular + "s"
}

func Singularize(plural string) string {
	if strings.HasSuffix(plural, "ies") {
		return strings.TrimSuffix(plural, "ies") + "y"
	}
	return strings.TrimSuffix(plural, "s")
}
