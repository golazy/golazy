package lazyassets

import (
	"errors"
	"fmt"
)

var (
	errNoHash = errors.New("path without hash")
)

type ErrNotFound string

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("file %s not found", string(e))
}
