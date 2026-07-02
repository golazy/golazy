package lazystorage

import (
	"errors"
	"time"
)

// ErrPreconditionFailed reports that a conditional storage write did not match
// the current object state.
var ErrPreconditionFailed = errors.New("lazystorage: precondition failed")

// ExpiresIn requests a URL or token that expires after Duration.
type ExpiresIn struct {
	Duration time.Duration
}

// ExpiresAt requests a URL or token that expires at Time.
type ExpiresAt struct {
	Time time.Time
}

// Public requests public access when the backend supports access policy.
type Public struct{}

// Private requests private access when the backend supports access policy.
type Private struct{}

// ContentType sets or requests a content type.
type ContentType struct {
	Value string
}

// CacheControl sets or requests a Cache-Control policy.
type CacheControl struct {
	Value string
}

// ContentDisposition sets or requests a Content-Disposition policy.
type ContentDisposition struct {
	Value string
}

// IfAbsent requests that Put succeeds only when the object does not exist.
type IfAbsent struct{}

// IfETag requests that Put succeeds only when the current object ETag matches
// Value.
type IfETag struct {
	Value string
}

// DownloadName requests a download filename for generated URLs.
type DownloadName struct {
	Filename string
}

// Take removes the first option assignable to T and returns it with the
// remaining options. It is useful for director implementations that consume
// recognized options and pass unknown ones downstream.
func Take[T any](options []any) (T, []any, bool) {
	var zero T
	for index, option := range options {
		value, ok := option.(T)
		if !ok {
			continue
		}
		remaining := make([]any, 0, len(options)-1)
		remaining = append(remaining, options[:index]...)
		remaining = append(remaining, options[index+1:]...)
		return value, remaining, true
	}
	return zero, options, false
}
