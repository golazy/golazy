package lazyassets

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const (
	abbrHash = 12
)

type hash [sha256.Size]byte

func newHash(data []byte) hash {
	h := sha256.Sum256(data)
	return hash(h)
}

func (h hash) Etag() string {
	return h.String()
}

func (h hash) Base64() string {
	return base64.StdEncoding.EncodeToString(h[:])
}

func (h hash) Integrity() string {
	return "sha256-" + h.Base64()
}

func (h hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

func (h hash) Short() string {
	s := h.String()
	if len(string(s)) > abbrHash {
		return string(s)[:abbrHash]
	}
	return string(s)
}

func (h hash) Zero() bool {
	for _, b := range h {
		if b != 0 {
			return false
		}
	}
	return true
}
