package lazyapp

import (
	"net/url"
	"strings"
	"time"
)

func absoluteURL(base, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if parsed, err := url.Parse(path); err == nil && parsed.IsAbs() {
		return path
	}
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func latestTime(left, right time.Time) time.Time {
	if right.IsZero() || (!left.IsZero() && !right.After(left)) {
		return left
	}
	return right
}
