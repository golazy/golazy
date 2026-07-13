package pg

import (
	_ "embed"
	"strings"

	"golazy.dev/lazyaddon"
)

//go:embed lazyaddon.toml
var addonManifestData []byte

var addonManifest = lazyaddon.MustParseManifest(addonManifestData)

// AddonDefinition returns a copy of the named definition from this module's
// embedded lazyaddon.toml manifest. Add-on packages use it to keep runtime
// registration aligned with the distribution manifest.
func AddonDefinition(id string) (lazyaddon.Definition, bool) {
	id = strings.TrimSpace(id)
	for _, definition := range addonManifest.Addons {
		if definition.ID != id {
			continue
		}
		definition.Requires = append([]string(nil), definition.Requires...)
		definition.Optional = append([]string(nil), definition.Optional...)
		definition.Conflicts = append([]string(nil), definition.Conflicts...)
		if definition.Version == "" {
			definition.Version = addonManifest.Package.Version
		}
		return definition, true
	}
	return lazyaddon.Definition{}, false
}
