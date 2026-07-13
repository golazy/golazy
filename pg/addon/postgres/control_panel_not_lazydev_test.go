//go:build !lazydev

package postgres_test

import (
	"testing"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/pg/addon/postgres"
)

func TestControlPlaneContributionIsExcludedOutsideLazyDev(t *testing.T) {
	scope, err := lazyaddon.Resolve(lazyaddon.Select(postgres.AddonID))
	if err != nil {
		t.Fatal(err)
	}
	if scope.HasCallbacks(lazyapp.ControlPlaneHook.ID()) {
		t.Fatal("production build includes the PostgreSQL lazydev control-plane callback")
	}
}
