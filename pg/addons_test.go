package pg

import (
	"reflect"
	"testing"
)

func TestAddonDefinitionReadsEmbeddedManifestAndReturnsCopies(t *testing.T) {
	jobs, ok := AddonDefinition("postgres/jobs")
	if !ok {
		t.Fatal("postgres/jobs definition is missing")
	}
	if got, want := jobs.Version, addonManifest.Package.Version; got != want {
		t.Fatalf("version = %q, want embedded package version %q", got, want)
	}
	if got, want := jobs.Requires, []string{"postgres@" + jobs.Version}; !reflect.DeepEqual(got, want) {
		t.Fatalf("requirements = %v, want %v", got, want)
	}

	jobs.Requires[0] = "changed"
	fresh, ok := AddonDefinition("postgres/jobs")
	if !ok {
		t.Fatal("postgres/jobs definition disappeared")
	}
	if got, want := fresh.Requires, []string{"postgres@" + fresh.Version}; !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh requirements = %v, want %v", got, want)
	}
	if _, ok := AddonDefinition("missing"); ok {
		t.Fatal("missing add-on definition was found")
	}
}
