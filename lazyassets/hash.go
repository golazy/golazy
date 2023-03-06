package lazyassets

import (
	"crypto/sha512"
	"encoding/base64"
	"fmt"
)

const (
	AbbrHash = 12
)

type Hash [sha512.Size384]byte

func NewHash(data []byte) Hash {
	h := sha512.Sum384(data)
	return Hash(h)
}

func (h Hash) Base64() string {
	return base64.StdEncoding.EncodeToString(h[:])
}

func (h Hash) Integrity() string {
	return "sha384-" + h.Base64()
}

func (h Hash) String() string {
	return fmt.Sprintf("%x", h[:])
}

func (h Hash) Short() string {
	s := h.String()
	if len(string(s)) > AbbrHash {
		return string(s)[:AbbrHash]
	}
	return string(s)
}

func (s Hash) Zero() bool {
	for _, b := range s {
		if b != 0 {
			return false
		}
	}
	return true
}
