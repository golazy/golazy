package lazyaddon

import (
	"reflect"
	"strings"
	"testing"
)

func TestManifestDependenciesResolveInDependencyOrder(t *testing.T) {
	manifest := MustParseManifest([]byte(`
manifest = 1

[package]
id = "golazy/postgres"
version = "v1.2.0"

[[addons]]
id = "postgres"
description = "pool"

[[addons]]
id = "postgres/jobs"
description = "jobs"
requires = ["postgres"]
	`))
	catalog := NewCatalog()
	if _, err := catalog.RegisterPackage(Package{Manifest: manifest}); err != nil {
		t.Fatal(err)
	}
	scope, err := catalog.Resolve(Select("postgres/jobs"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := scope.Addons(), []string{"postgres", "postgres/jobs"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("addons = %v, want %v", got, want)
	}
}

func TestResolveRejectsDependencyCycle(t *testing.T) {
	catalog := NewCatalog()
	for _, definition := range []Definition{
		{ID: "a", Version: "v1", Requires: []string{"b"}},
		{ID: "b", Version: "v1", Requires: []string{"a"}},
	} {
		if _, err := catalog.Register(definition); err != nil {
			t.Fatal(err)
		}
	}
	_, err := catalog.Resolve(Select("a"))
	if err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("Resolve error = %v, want dependency cycle", err)
	}
}

func TestResolveRejectsSelectedVersionMismatch(t *testing.T) {
	catalog := NewCatalog()
	if _, err := catalog.Register(Definition{ID: "example", Version: "v1.1.0"}); err != nil {
		t.Fatal(err)
	}
	_, err := catalog.Resolve(Selection{Addons: []Use{{ID: "example", Version: "v1.0.0"}}})
	if err == nil || !strings.Contains(err.Error(), `registered version "v1.1.0" does not match selected version "v1.0.0"`) {
		t.Fatalf("Resolve error = %v, want selected-version mismatch", err)
	}
}

func TestRegisterRequiresStableIdentityAndVersion(t *testing.T) {
	tests := []struct {
		definition Definition
		contains   string
	}{
		{definition: Definition{ID: "Postgres", Version: "v1"}, contains: "invalid character"},
		{definition: Definition{ID: "postgres/jobs"}, contains: "version is required"},
	}
	for _, test := range tests {
		catalog := NewCatalog()
		_, err := catalog.Register(test.definition)
		if err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Fatalf("Register(%+v) error = %v, want %q", test.definition, err, test.contains)
		}
	}
}

func TestRegisterRejectsInvalidPathLikeIDs(t *testing.T) {
	for _, id := range []string{"/seo", "seo/", "seo//meta", "seo/.", "seo/.."} {
		t.Run(id, func(t *testing.T) {
			catalog := NewCatalog()
			if _, err := catalog.Register(Definition{ID: id, Version: "v1"}); err == nil {
				t.Fatalf("Register(%q) succeeded", id)
			}
		})
	}
}

func TestTypedCallbacksAreSelectedOrderedAndIsolated(t *testing.T) {
	type event struct{ Calls []string }
	catalog := NewCatalog()
	hook := DefineHookIn[event](catalog, "test/configure", 1)
	registrations := map[string]Registration{}
	for _, definition := range []Definition{{ID: "a", Version: "v1"}, {ID: "b", Version: "v1"}} {
		registration, err := catalog.Register(definition)
		if err != nil {
			t.Fatal(err)
		}
		registrations[definition.ID] = registration
	}
	if err := OnIn(catalog, registrations["b"], hook, CallbackOptions{ID: "b/run", After: []string{"a/run"}}, func(event *event) error {
		event.Calls = append(event.Calls, "b")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := OnIn(catalog, registrations["a"], hook, CallbackOptions{ID: "a/run"}, func(event *event) error {
		event.Calls = append(event.Calls, "a")
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	scopeA, err := catalog.Resolve(Select("a"))
	if err != nil {
		t.Fatal(err)
	}
	eventA := event{}
	if err := Run(scopeA, hook, &eventA); err != nil {
		t.Fatal(err)
	}
	if got, want := eventA.Calls, []string{"a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("app A calls = %v, want %v", got, want)
	}

	scopeBoth, err := catalog.Resolve(Select("b", "a"))
	if err != nil {
		t.Fatal(err)
	}
	eventBoth := event{}
	if err := Run(scopeBoth, hook, &eventBoth); err != nil {
		t.Fatal(err)
	}
	if got, want := eventBoth.Calls, []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("app both calls = %v, want %v", got, want)
	}
}

func TestCallbacksRunInTransitiveDependencyOrderByDefault(t *testing.T) {
	type event struct{ Calls []string }
	catalog := NewCatalog()
	hook := DefineHookIn[event](catalog, "test/dependency-order", 1)
	registrations := map[string]Registration{}
	for _, definition := range []Definition{
		{ID: "base", Version: "v1"},
		{ID: "middle", Version: "v1", Requires: []string{"base"}},
		{ID: "feature", Version: "v1", Requires: []string{"middle"}},
	} {
		registration, err := catalog.Register(definition)
		if err != nil {
			t.Fatal(err)
		}
		registrations[definition.ID] = registration
	}
	// The IDs intentionally sort in the opposite order. The dependency graph,
	// not callback lexicography or registration order, controls execution.
	for _, callback := range []struct {
		addon string
		id    string
	}{
		{addon: "feature", id: "a-feature"},
		{addon: "base", id: "z-base"},
	} {
		callback := callback
		if err := OnIn(catalog, registrations[callback.addon], hook, CallbackOptions{ID: callback.id}, func(event *event) error {
			event.Calls = append(event.Calls, callback.addon)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	scope, err := catalog.Resolve(Select("feature"))
	if err != nil {
		t.Fatal(err)
	}
	gotEvent := event{}
	if err := Run(scope, hook, &gotEvent); err != nil {
		t.Fatal(err)
	}
	if got, want := gotEvent.Calls, []string{"base", "feature"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
}

func TestCallbackOrderingCannotInvertAddonDependency(t *testing.T) {
	type event struct{ Calls int }
	catalog := NewCatalog()
	hook := DefineHookIn[event](catalog, "test/dependency-cycle", 1)
	base, err := catalog.Register(Definition{ID: "base", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	feature, err := catalog.Register(Definition{ID: "feature", Version: "v1", Requires: []string{"base"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := OnIn(catalog, base, hook, CallbackOptions{ID: "base/run"}, func(event *event) error {
		event.Calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := OnIn(catalog, feature, hook, CallbackOptions{ID: "feature/run", Before: []string{"base/run"}}, func(event *event) error {
		event.Calls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	scope, err := catalog.Resolve(Select("feature"))
	if err != nil {
		t.Fatal(err)
	}
	gotEvent := event{}
	err = Run(scope, hook, &gotEvent)
	if err == nil || !strings.Contains(err.Error(), "ordering cycle") {
		t.Fatalf("Run error = %v, want ordering cycle", err)
	}
	if gotEvent.Calls != 0 {
		t.Fatalf("callbacks ran before dependency order validation: %d", gotEvent.Calls)
	}
}

func TestCallbackCycleFailsBeforeInvocation(t *testing.T) {
	type event struct{ Calls int }
	catalog := NewCatalog()
	hook := DefineHookIn[event](catalog, "test/cycle", 1)
	registration, err := catalog.Register(Definition{ID: "a", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	for _, callback := range []struct {
		id     string
		before string
	}{
		{id: "a/one", before: "a/two"},
		{id: "a/two", before: "a/one"},
	} {
		if err := OnIn(catalog, registration, hook, CallbackOptions{ID: callback.id, Before: []string{callback.before}}, func(event *event) error {
			event.Calls++
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	scope, err := catalog.Resolve(Select("a"))
	if err != nil {
		t.Fatal(err)
	}
	gotEvent := event{}
	err = Run(scope, hook, &gotEvent)
	if err == nil || !strings.Contains(err.Error(), "ordering cycle") {
		t.Fatalf("Run error = %v, want ordering cycle", err)
	}
	if gotEvent.Calls != 0 {
		t.Fatalf("callbacks ran before validation: %d", gotEvent.Calls)
	}
}

func TestCapabilitiesArePerScope(t *testing.T) {
	catalog := NewCatalog()
	registration, err := catalog.Register(Definition{ID: "database", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	capability := DefineCapabilityIn[string](catalog, registration, "test/database", 1)
	first, err := catalog.Resolve(Select("database"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Resolve(Select("database"))
	if err != nil {
		t.Fatal(err)
	}
	if err := Provide(first, registration, capability, "first"); err != nil {
		t.Fatal(err)
	}
	if got, err := Require(first, capability); err != nil || got != "first" {
		t.Fatalf("Require(first) = %q, %v", got, err)
	}
	if _, err := Require(second, capability); err == nil {
		t.Fatal("Require(second) succeeded before a provider was registered")
	}
}

func TestCallbackRegistrationRejectsForgedAndForeignOwners(t *testing.T) {
	type event struct{}
	catalog := NewCatalog()
	hook := DefineHookIn[event](catalog, "test/owner", 1)
	owner, err := catalog.Register(Definition{ID: "owner", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	callback := func(*event) error { return nil }

	forged := Registration{catalog: catalog, id: owner.id, token: &registrationToken{}}
	if err := OnIn(catalog, forged, hook, CallbackOptions{ID: "forged"}, callback); err == nil || !strings.Contains(err.Error(), "does not own") {
		t.Fatalf("OnIn(forged) error = %v, want ownership error", err)
	}
	if err := OnIn(catalog, Registration{}, hook, CallbackOptions{ID: "zero"}, callback); err == nil || !strings.Contains(err.Error(), "registration is invalid") {
		t.Fatalf("OnIn(zero) error = %v, want invalid registration", err)
	}

	foreignCatalog := NewCatalog()
	foreign, err := foreignCatalog.Register(Definition{ID: "owner", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := OnIn(catalog, foreign, hook, CallbackOptions{ID: "foreign"}, callback); err == nil || !strings.Contains(err.Error(), "different catalog") {
		t.Fatalf("OnIn(foreign) error = %v, want catalog mismatch", err)
	}
	foreignHook := DefineHookIn[event](foreignCatalog, "test/owner", 1)
	if err := OnIn(catalog, owner, foreignHook, CallbackOptions{ID: "foreign-hook"}, callback); err == nil || !strings.Contains(err.Error(), "different catalog") {
		t.Fatalf("OnIn(foreign hook) error = %v, want catalog mismatch", err)
	}

	if err := OnIn(catalog, owner, hook, CallbackOptions{ID: "valid"}, callback); err != nil {
		t.Fatalf("OnIn(valid) error = %v", err)
	}
}

func TestCapabilityPublicationRequiresDefiningOwner(t *testing.T) {
	catalog := NewCatalog()
	owner, err := catalog.Register(Definition{ID: "database", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	attacker, err := catalog.Register(Definition{ID: "attacker", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	capability := DefineCapabilityIn[string](catalog, owner, "test/owned-database", 1)
	scope, err := catalog.Resolve(Select("database", "attacker"))
	if err != nil {
		t.Fatal(err)
	}

	if err := Provide(scope, attacker, capability, "claimed"); err == nil || !strings.Contains(err.Error(), "does not own capability") {
		t.Fatalf("Provide(attacker) error = %v, want capability ownership error", err)
	}
	forged := Registration{catalog: catalog, id: owner.id, token: &registrationToken{}}
	if err := Provide(scope, forged, capability, "forged"); err == nil || !strings.Contains(err.Error(), "does not own selected add-on") {
		t.Fatalf("Provide(forged) error = %v, want selected add-on ownership error", err)
	}
	if err := Provide(scope, owner, capability, "owned"); err != nil {
		t.Fatalf("Provide(owner) error = %v", err)
	}
	if got, err := Require(scope, capability); err != nil || got != "owned" {
		t.Fatalf("Require(scope) = %q, %v, want owned", got, err)
	}
}

func TestCapabilityPublicationRejectsCatalogMismatchedHandle(t *testing.T) {
	firstCatalog := NewCatalog()
	firstOwner, err := firstCatalog.Register(Definition{ID: "database", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	firstCapability := DefineCapabilityIn[string](firstCatalog, firstOwner, "test/catalog-database", 1)

	secondCatalog := NewCatalog()
	secondOwner, err := secondCatalog.Register(Definition{ID: "database", Version: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	secondCapability := DefineCapabilityIn[string](secondCatalog, secondOwner, "test/catalog-database", 1)
	secondScope, err := secondCatalog.Resolve(Select("database"))
	if err != nil {
		t.Fatal(err)
	}

	if err := Provide(secondScope, firstOwner, firstCapability, "wrong catalog"); err == nil || !strings.Contains(err.Error(), "different catalog") {
		t.Fatalf("Provide(foreign catalog) error = %v, want catalog mismatch", err)
	}
	if err := Provide(secondScope, secondOwner, firstCapability, "foreign descriptor"); err == nil || !strings.Contains(err.Error(), "does not own capability") {
		t.Fatalf("Provide(foreign descriptor) error = %v, want descriptor ownership error", err)
	}
	if err := Provide(secondScope, secondOwner, secondCapability, "second"); err != nil {
		t.Fatalf("Provide(second) error = %v", err)
	}
	if _, err := Require(secondScope, firstCapability); err == nil || !strings.Contains(err.Error(), "contract mismatch") {
		t.Fatalf("Require(foreign descriptor) error = %v, want contract mismatch", err)
	}
}
