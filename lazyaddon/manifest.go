package lazyaddon

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// Manifest is the runtime-relevant subset of lazyaddon.toml. Distribution and
// installer-specific contribution metadata remains available in Raw.
type Manifest struct {
	Schema  int
	Package PackageInfo
	Addons  []Definition
	Raw     []byte
	Digest  string
}

// PackageInfo identifies one versioned add-on distribution.
type PackageInfo struct {
	ID      string
	Version string
}

// ParseManifest parses the core package and add-on dependency declarations
// from lazyaddon.toml without adding a TOML dependency to application binaries.
func ParseManifest(data []byte) (Manifest, error) {
	manifest := Manifest{Raw: append([]byte(nil), data...)}
	sum := sha256.Sum256(data)
	manifest.Digest = "sha256:" + hex.EncodeToString(sum[:])

	section := ""
	addonIndex := -1
	for index, raw := range strings.Split(string(data), "\n") {
		lineNumber := index + 1
		line := strings.TrimSpace(stripManifestComment(raw))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[[") {
			if line != "[[addons]]" {
				continue
			}
			manifest.Addons = append(manifest.Addons, Definition{})
			addonIndex = len(manifest.Addons) - 1
			section = "addons"
			continue
		}
		if strings.HasPrefix(line, "[") {
			if !strings.HasSuffix(line, "]") {
				return Manifest{}, fmt.Errorf("lazyaddon: manifest line %d: malformed table", lineNumber)
			}
			section = strings.TrimSpace(line[1 : len(line)-1])
			addonIndex = -1
			continue
		}
		key, rawValue, ok := strings.Cut(line, "=")
		if !ok {
			return Manifest{}, fmt.Errorf("lazyaddon: manifest line %d: expected key = value", lineNumber)
		}
		key = strings.TrimSpace(key)
		rawValue = strings.TrimSpace(rawValue)
		var err error
		switch section {
		case "":
			if key == "manifest" || key == "schema" {
				manifest.Schema, err = strconv.Atoi(rawValue)
			}
		case "package":
			switch key {
			case "id":
				manifest.Package.ID, err = parseManifestString(rawValue)
			case "version":
				manifest.Package.Version, err = parseManifestString(rawValue)
			}
		case "addons":
			if addonIndex < 0 {
				continue
			}
			addon := &manifest.Addons[addonIndex]
			switch key {
			case "id":
				addon.ID, err = parseManifestString(rawValue)
			case "version":
				addon.Version, err = parseManifestString(rawValue)
			case "description":
				addon.Description, err = parseManifestString(rawValue)
			case "requires":
				addon.Requires, err = parseManifestArray(rawValue)
			case "optional":
				addon.Optional, err = parseManifestArray(rawValue)
			case "conflicts":
				addon.Conflicts, err = parseManifestArray(rawValue)
			}
		}
		if err != nil {
			return Manifest{}, fmt.Errorf("lazyaddon: manifest line %d: %w", lineNumber, err)
		}
	}
	if manifest.Schema == 0 {
		manifest.Schema = 1
	}
	// Package-local dependencies default to the package release. This keeps a
	// dependent add-on from silently resolving against a newer incompatible
	// sibling while preserving the concise requires = ["base"] form.
	for addonIndex := range manifest.Addons {
		for requirementIndex, requirement := range manifest.Addons[addonIndex].Requires {
			if _, version := splitRequirement(requirement); version == "" {
				manifest.Addons[addonIndex].Requires[requirementIndex] = strings.TrimSpace(requirement) + "@" + manifest.Package.Version
			}
		}
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// Validate validates core manifest identity and dependency declarations.
func (manifest Manifest) Validate() error {
	if manifest.Schema != 1 {
		return fmt.Errorf("lazyaddon: unsupported manifest schema %d", manifest.Schema)
	}
	if strings.TrimSpace(manifest.Package.ID) == "" {
		return fmt.Errorf("lazyaddon: manifest package ID is required")
	}
	if strings.TrimSpace(manifest.Package.Version) == "" {
		return fmt.Errorf("lazyaddon: manifest package version is required")
	}
	if len(manifest.Addons) == 0 {
		return fmt.Errorf("lazyaddon: manifest must declare at least one add-on")
	}
	seen := map[string]bool{}
	for _, definition := range manifest.Addons {
		definition = normalizeDefinition(definition)
		if definition.Version == "" {
			definition.Version = manifest.Package.Version
		}
		if err := validateDefinition(definition); err != nil {
			return err
		}
		if seen[definition.ID] {
			return fmt.Errorf("lazyaddon: manifest add-on %q is duplicated", definition.ID)
		}
		seen[definition.ID] = true
	}
	return nil
}

// MustParseManifest parses data and panics on failure.
func MustParseManifest(data []byte) Manifest {
	manifest, err := ParseManifest(data)
	if err != nil {
		panic(err)
	}
	return manifest
}

func stripManifestComment(line string) string {
	quoted := false
	escaped := false
	for index, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quoted {
			escaped = true
			continue
		}
		if r == '"' {
			quoted = !quoted
			continue
		}
		if r == '#' && !quoted {
			return line[:index]
		}
	}
	return line
}

func parseManifestString(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := strconv.Unquote(value)
	if err != nil {
		return "", fmt.Errorf("invalid quoted string %q", value)
	}
	return strings.TrimSpace(parsed), nil
}

func parseManifestArray(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 2 || value[0] != '[' || value[len(value)-1] != ']' {
		return nil, fmt.Errorf("invalid string array %q", value)
	}
	value = strings.TrimSpace(value[1 : len(value)-1])
	if value == "" {
		return nil, nil
	}
	var out []string
	for len(value) > 0 {
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(value, "\"") {
			return nil, fmt.Errorf("invalid string array %q", value)
		}
		end := 1
		escaped := false
		for ; end < len(value); end++ {
			if escaped {
				escaped = false
				continue
			}
			if value[end] == '\\' {
				escaped = true
				continue
			}
			if value[end] == '"' {
				break
			}
		}
		if end >= len(value) {
			return nil, fmt.Errorf("unterminated string array value")
		}
		parsed, err := strconv.Unquote(value[:end+1])
		if err != nil {
			return nil, err
		}
		out = append(out, strings.TrimSpace(parsed))
		value = strings.TrimSpace(value[end+1:])
		if value == "" {
			break
		}
		if value[0] != ',' {
			return nil, fmt.Errorf("expected comma in string array")
		}
		value = value[1:]
	}
	return normalizeIDs(out), nil
}
