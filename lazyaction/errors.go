package lazyaction

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrNotAuthorized = errors.New("not authorized")
)
