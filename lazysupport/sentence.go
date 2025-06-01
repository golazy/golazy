package lazysupport

import "strings"

// ToSentence convert a slice of strings into a sentence
// with the last_join string between the last two parts.
func ToSentence(last_join string, parts ...string) string {
	if last_join == "" {
		last_join = "and"
	}
	l := len(parts)
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " " + last_join + " " + parts[1]
	default:
		return strings.Join(parts[0:l-1], ", ") + " " + last_join + " " + parts[l-1]
	}

}
