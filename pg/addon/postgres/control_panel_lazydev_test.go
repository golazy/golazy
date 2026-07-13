//go:build lazydev

package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
)

func TestLazyDevPanelRegistersOwnedSafePoolStatus(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://panel_user:super-secret@database.internal:5432/panel_test?sslmode=disable")
	previousPing := pingPostgresPool
	deadlineSeen := false
	pingCalls := 0
	pingPostgresPool = func(ctx context.Context, _ *pgxpool.Pool) error {
		pingCalls++
		_, deadlineSeen = ctx.Deadline()
		return errors.New("dial database.internal with password super-secret")
	}
	t.Cleanup(func() { pingPostgresPool = previousPing })

	scope, dependencies := lazyDevAddonScope(t)
	if !scope.HasCallbacks(lazyapp.ControlPlaneHook.ID()) {
		t.Fatal("lazydev build did not register the PostgreSQL control-plane callback")
	}
	controlPlane := lazycontrolplane.New(lazycontrolplane.Config{})
	if err := lazyaddon.Run(scope, lazyapp.ControlPlaneHook, &lazyapp.ControlPlaneEvent{
		Context:      dependencies.Context(),
		ControlPlane: controlPlane,
		Addons:       scope,
	}); err != nil {
		t.Fatal(err)
	}

	panels := controlPlane.Panels()
	if len(panels) != 1 {
		t.Fatalf("panels = %#v, want one PostgreSQL panel", panels)
	}
	panel := panels[0]
	if panel.ID != lazyDevPanelID || panel.Owner != AddonID || panel.EndpointID != lazyDevEndpointID {
		t.Fatalf("panel = %#v, want owned PostgreSQL descriptor", panel)
	}
	if panel.Method != http.MethodGet || panel.Path != lazyDevPath {
		t.Fatalf("panel endpoint = %s %s, want GET %s", panel.Method, panel.Path, lazyDevPath)
	}
	if len(panel.Actions) != 1 {
		t.Fatalf("panel actions = %#v, want one ping action", panel.Actions)
	}
	action := panel.Actions[0]
	if action.ID != "ping" || action.EndpointID != lazyDevPingEndpointID || action.Method != http.MethodPost || action.Path != lazyDevPingPath {
		t.Fatalf("panel action = %#v, want owned POST ping descriptor", action)
	}

	recorder := httptest.NewRecorder()
	controlPlane.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, lazyDevPath, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := recorder.Body.String()
	var response lazyDevPoolResponse
	if err := json.NewDecoder(strings.NewReader(body)).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Healthy || response.Status != "unavailable" {
		t.Fatalf("pool status = %#v, want sanitized unavailable status", response)
	}
	if !deadlineSeen {
		t.Fatal("pool health check ran without a request timeout")
	}
	if response.Pool.MaxConnections <= 0 {
		t.Fatalf("max connections = %d, want configured pool statistics", response.Pool.MaxConnections)
	}
	for _, secret := range []string{"super-secret", "panel_user", "database.internal", "panel_test", "DATABASE_URL"} {
		if strings.Contains(body, secret) {
			t.Fatalf("response leaked %q: %s", secret, body)
		}
	}

	recorder = httptest.NewRecorder()
	controlPlane.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, lazyDevPingPath, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("ping action status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if pingCalls != 2 {
		t.Fatalf("pool ping calls = %d, want one display ping and one action ping", pingCalls)
	}
	for _, secret := range []string{"super-secret", "panel_user", "database.internal", "panel_test", "DATABASE_URL"} {
		if strings.Contains(recorder.Body.String(), secret) {
			t.Fatalf("ping action response leaked %q: %s", secret, recorder.Body.String())
		}
	}
}

func lazyDevAddonScope(t *testing.T) (*lazyaddon.Scope, *lazydeps.Scope) {
	t.Helper()
	scope, err := lazyaddon.Resolve(lazyaddon.Select(AddonID))
	if err != nil {
		t.Fatal(err)
	}
	dependencies := lazydeps.New(context.Background())
	t.Cleanup(func() {
		if err := dependencies.Shutdown(context.Background(), "test finished"); err != nil {
			t.Errorf("shutdown dependencies: %v", err)
		}
	})
	if err := lazyaddon.Run(scope, lazyapp.DependenciesHook, &lazyapp.DependenciesEvent{
		Context:      context.Background(),
		Dependencies: dependencies,
		Addons:       scope,
	}); err != nil {
		t.Fatal(err)
	}
	return scope, dependencies
}
