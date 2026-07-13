package lazyapp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
)

func TestSelectedAddonRunsLifecycleHooksPerApp(t *testing.T) {
	const addonID = "lazyapp-test/lifecycle"
	registration := lazyaddon.MustRegisterDefinition(lazyaddon.Definition{ID: addonID, Version: "v1"})
	var dependencyRuns atomic.Int64
	lazyaddon.MustOn(registration, DependenciesHook, lazyaddon.CallbackOptions{ID: "dependencies"}, func(event *DependenciesEvent) error {
		dependencyRuns.Add(1)
		_, err := lazydeps.Service(event.Dependencies, "addon-test", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
			return ctx, "ready", nil, nil
		})
		return err
	})
	lazyaddon.MustOn(registration, RoutesHook, lazyaddon.CallbackOptions{ID: "routes"}, func(event *RoutesEvent) error {
		event.Router.HandleFunc(http.MethodGet, "/addon-test", func(w http.ResponseWriter, _ *http.Request) error {
			_, _ = w.Write([]byte("addon"))
			return nil
		})
		return nil
	})
	lazyaddon.MustOn(registration, ControlPlaneHook, lazyaddon.CallbackOptions{ID: "controlplane"}, func(event *ControlPlaneEvent) error {
		if err := event.ControlPlane.Register(lazycontrolplane.Endpoint{
			ID:          "lazyapp-test.endpoint",
			Owner:       addonID,
			Pattern:     "GET /addon-test/status",
			Description: "Add-on test status",
			Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("ready"))
			}),
		}); err != nil {
			return err
		}
		return event.ControlPlane.RegisterPanel(lazycontrolplane.Panel{
			ID:          "lazyapp-test.panel",
			Owner:       addonID,
			Title:       "Test Add-on",
			Description: "Test add-on functionality",
			EndpointID:  "lazyapp-test.endpoint",
		})
	})

	without := New(Config{Name: "without-addon"})
	if without.Addons.Has(addonID) {
		t.Fatal("unselected app activated add-on")
	}
	if dependencyRuns.Load() != 0 {
		t.Fatalf("dependency runs without selection = %d", dependencyRuns.Load())
	}

	with := New(Config{Name: "with-addon", Addons: lazyaddon.Select(addonID)})
	if !with.Addons.Has(addonID) {
		t.Fatal("selected app is missing add-on")
	}
	if dependencyRuns.Load() != 1 {
		t.Fatalf("dependency runs = %d, want 1", dependencyRuns.Load())
	}
	appHandler, controlHandler := with.handlersForListen(defaultListenAddr, ":9090", true)
	if controlHandler == nil {
		t.Fatal("add-on control-plane handler is nil")
	}

	response := httptest.NewRecorder()
	appHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/addon-test", nil))
	if response.Code != http.StatusOK || response.Body.String() != "addon" {
		t.Fatalf("add-on route = %d %q", response.Code, response.Body.String())
	}
	if with.ControlPlane == nil || !with.ControlPlane.HandlesPath("/addon-test/status") {
		t.Fatal("add-on control-plane endpoint was not registered")
	}
	response = httptest.NewRecorder()
	controlHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/addon-test/status", nil))
	if response.Code != http.StatusOK || response.Body.String() != "ready" {
		t.Fatalf("control-plane endpoint = %d %q", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	controlHandler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, lazycontrolplane.PanelsPath, nil))
	var panels lazycontrolplane.PanelsResponse
	if err := json.NewDecoder(response.Body).Decode(&panels); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(panels.Panels) != 1 || panels.Panels[0].ID != "lazyapp-test.panel" {
		t.Fatalf("add-on panels = %d %#v", response.Code, panels)
	}
	if err := with.ControlPlane.Register(lazycontrolplane.Endpoint{
		ID:          "late",
		Owner:       addonID,
		Pattern:     "GET /late",
		Description: "late endpoint",
		Handler:     http.NotFoundHandler(),
	}); !errors.Is(err, lazycontrolplane.ErrSealed) {
		t.Fatalf("late control-plane registration error = %v, want ErrSealed", err)
	}
}

func TestAddonDependenciesAreAvailableToApplicationServices(t *testing.T) {
	const addonID = "lazyapp-test/dependency-provider"
	type contextKey struct{}
	registration := lazyaddon.MustRegisterDefinition(lazyaddon.Definition{ID: addonID, Version: "v1"})
	lazyaddon.MustOn(registration, DependenciesHook, lazyaddon.CallbackOptions{ID: "provider"}, func(event *DependenciesEvent) error {
		_, err := lazydeps.Service(event.Dependencies, "addon-provider", func(ctx context.Context) (context.Context, string, error, context.CancelFunc) {
			return context.WithValue(ctx, contextKey{}, "from-addon"), "from-addon", nil, nil
		})
		return err
	})

	applicationSawProvider := false
	app := New(Config{
		Name:   "addon-dependency-order",
		Addons: lazyaddon.Select(addonID),
		Dependencies: func(scope *lazydeps.Scope) error {
			applicationSawProvider = scope.Context().Value(contextKey{}) == "from-addon"
			return nil
		},
	})
	if !applicationSawProvider {
		t.Fatal("application dependencies ran before the selected add-on provider")
	}
	if got := app.Context.Value(contextKey{}); got != "from-addon" {
		t.Fatalf("application context provider value = %v, want from-addon", got)
	}
}
