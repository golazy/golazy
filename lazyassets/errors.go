package lazyassets

import (
	"errors"
)

var (
	errNoHash   = errors.New("path without hash")
	ErrNotFound = errors.New("not found")
)
