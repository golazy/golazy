package lazyassets

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"path"
	"strings"
)

type hash [sha256.Size]byte

func newHash(data []byte) hash {
	return hash(sha256.Sum256(data))
}

func (h hash) Hex() string {
	return fmt.Sprintf("%x", h[:])
}

func (h hash) Short(length int) string {
	hex := h.Hex()
	if length > len(hex) {
		return hex
	}
	return hex[:length]
}

func (h hash) ETag() string {
	return fmt.Sprintf("%q", h.Hex())
}

func (h hash) Integrity() string {
	return "sha256-" + base64.StdEncoding.EncodeToString(h[:])
}

func withHash(assetPath, digest string) string {
	ext := path.Ext(assetPath)
	if ext == "" {
		return assetPath + "-" + digest
	}
	return strings.TrimSuffix(assetPath, ext) + "-" + digest + ext
}

func etagMatches(header, etag string) bool {
	header = strings.TrimSpace(header)
	if header == "" || etag == "" {
		return false
	}
	if header == "*" {
		return true
	}
	for _, candidate := range splitETagList(header) {
		if weakETagValue(candidate) == weakETagValue(etag) {
			return true
		}
	}
	return false
}

func weakETagValue(etag string) string {
	etag = strings.TrimSpace(etag)
	if strings.HasPrefix(etag, "W/") || strings.HasPrefix(etag, "w/") {
		etag = strings.TrimSpace(etag[2:])
	}
	return etag
}

func splitETagList(header string) []string {
	var values []string
	start := 0
	inQuote := false
	escaped := false
	for index, char := range header {
		switch {
		case escaped:
			escaped = false
		case char == '\\' && inQuote:
			escaped = true
		case char == '"':
			inQuote = !inQuote
		case char == ',' && !inQuote:
			values = append(values, strings.TrimSpace(header[start:index]))
			start = index + 1
		}
	}
	values = append(values, strings.TrimSpace(header[start:]))
	return values
}
