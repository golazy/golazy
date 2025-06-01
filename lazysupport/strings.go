package lazysupport

import "strings"

type Strings Set[string]

func (s Strings) TrimPrefix(what string) (prefix, trimmed string) {
	for key := range s {
		if strings.HasPrefix(what, key) {
			return key, what[len(key):]
		}
	}
	return "", what
}

func (s Strings) HasPrefix(what string) bool {
	for key := range s {
		if strings.HasPrefix(what, key) {
			return true
		}
	}
	return false
}

func (s Strings) Set(what string) {
	Set[string](s).Set(what)
}
func (s Strings) Has(what string) bool {
	_, ok := s[what]
	return ok
}

func (s Strings) Slice() []string {
	return Set[string](s).Slice()
}

func NewStringSet(s ...string) Strings {
	return Strings(NewSet(s...))
}
