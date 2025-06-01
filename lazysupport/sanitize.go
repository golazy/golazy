package lazysupport

import "github.com/microcosm-cc/bluemonday"

// Sanitize sanitizes the input string using [bluemonday.UGCPolicy]
func Sanitize(input string) string {
	return bluemonday.UGCPolicy().Sanitize(input)
}

// StripTags strips all tags from the input string using [bluemonday.StrictPolicy]
func StripTags(input string) string {
	return bluemonday.StrictPolicy().Sanitize(input)
}
