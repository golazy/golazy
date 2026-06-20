package lazypath

import (
	"fmt"
	"net/url"
	"strings"
)

// URLParams appends query parameters to a generated route path.
type URLParams map[string]any

// SplitValues separates route parameter values from trailing path options.
func SplitValues(values []any) ([]any, URLParams) {
	if len(values) == 0 {
		return values, nil
	}
	if params, ok := values[len(values)-1].(URLParams); ok {
		return values[:len(values)-1], params
	}
	return values, nil
}

// AppendURLParams appends query parameters to path.
func AppendURLParams(path string, params URLParams) string {
	if len(params) == 0 {
		return path
	}

	values := url.Values{}
	for key, value := range params {
		if value == nil {
			continue
		}
		values.Set(key, fmt.Sprint(value))
	}
	encoded := values.Encode()
	if encoded == "" {
		return path
	}
	prefix, suffix := splitFragment(path)
	if strings.Contains(prefix, "?") {
		return prefix + "&" + encoded + suffix
	}
	return prefix + "?" + encoded + suffix
}

func splitFragment(path string) (string, string) {
	index := strings.Index(path, "#")
	if index < 0 {
		return path, ""
	}
	return path[:index], path[index:]
}
