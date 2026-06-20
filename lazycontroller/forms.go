package lazycontroller

import (
	"fmt"
	"net/http"

	"golazy.dev/lazyschema"
)

// Decode parses the current request form and decodes submitted fields into target.
func (b *Base) Decode(target any) error {
	if b.request == nil {
		return fmt.Errorf("controller request is not initialized")
	}
	if err := b.request.ParseForm(); err != nil {
		return Error(http.StatusBadRequest, err)
	}

	decoder := lazyschema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.ZeroEmpty(true)
	if err := decoder.Decode(target, b.request.PostForm); err != nil {
		return Error(http.StatusBadRequest, err)
	}
	return nil
}
