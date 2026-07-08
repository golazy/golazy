package lazyapp

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type cacheSizeBytes int64

func (s *cacheSizeBytes) UnmarshalText(text []byte) error {
	value, err := parseCacheSizeBytes(string(text))
	if err != nil {
		return err
	}
	*s = cacheSizeBytes(value)
	return nil
}

func parseCacheSizeBytes(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	index := 0
	for index < len(raw) && raw[index] >= '0' && raw[index] <= '9' {
		index++
	}
	if index == 0 {
		return 0, fmt.Errorf("cache size must start with a byte count")
	}
	value, err := strconv.ParseInt(raw[:index], 10, 64)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("cache size must be greater than zero")
	}
	unit := strings.ToLower(strings.TrimSpace(raw[index:]))
	multiplier, ok := cacheSizeUnitMultiplier(unit)
	if !ok {
		return 0, fmt.Errorf("unsupported cache size unit %q", unit)
	}
	if value > math.MaxInt64/multiplier {
		return 0, fmt.Errorf("cache size overflows int64")
	}
	return value * multiplier, nil
}

func cacheSizeUnitMultiplier(unit string) (int64, bool) {
	switch unit {
	case "", "b", "byte", "bytes":
		return 1, true
	case "k", "kb", "kib", "kilobyte", "kilobytes":
		return 1024, true
	case "m", "mb", "mib", "megabyte", "megabytes":
		return 1024 * 1024, true
	case "g", "gb", "gib", "gigabyte", "gigabytes":
		return 1024 * 1024 * 1024, true
	default:
		return 0, false
	}
}
