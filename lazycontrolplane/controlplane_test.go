package lazycontrolplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

var _ Builder = Config{}
var _ Builder = (*ControlPlane)(nil)
var _ Registrar = (*ControlPlane)(nil)

func TestEmptyConfigServesLiveAndReady(t *testing.T) {
	plane := New(Config{})

	for _, test := range []struct {
		path string
		body string
	}{
		{path: "/livez", body: "live\n"},
		{path: "/readyz", body: "ready\n"},
	} {
		response := httptest.NewRecorder()
		plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))

		if response.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d", test.path, response.Code, http.StatusOK)
		}
		if got := response.Body.String(); got != test.body {
			t.Fatalf("%s body = %q, want %q", test.path, got, test.body)
		}
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s Cache-Control = %q, want no-store", test.path, got)
		}
	}
}

func TestReadyzReportsFailedCheck(t *testing.T) {
	plane := New(Config{
		Readiness: []ReadinessCheck{{
			Name: "database",
			Check: func(context.Context) error {
				return errors.New("connection refused")
			},
		}},
	})

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if got := response.Body.String(); !strings.Contains(got, "not ready: database: connection refused") {
		t.Fatalf("body = %q, want failed check", got)
	}
}

func TestAddReadinessCheckAppendsReadyzCheck(t *testing.T) {
	plane := New(Config{})
	plane.AddReadinessCheck(ReadinessCheck{
		Name: "shutdown",
		Check: func(context.Context) error {
			return errors.New("application is draining")
		},
	})

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
	if got := response.Body.String(); !strings.Contains(got, "not ready: shutdown: application is draining") {
		t.Fatalf("body = %q, want added check failure", got)
	}
}

func TestMetricsIsOptional(t *testing.T) {
	plane := New(Config{})

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}

	plane = New(Config{
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "sample_metric 1\n")
		}),
	})
	response = httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("configured status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Body.String(); got != "sample_metric 1\n" {
		t.Fatalf("configured body = %q, want metric", got)
	}
}

func TestHandleRegistersCustomControlPlaneEndpoint(t *testing.T) {
	plane := New(Config{})
	plane.Handle("POST /_golazy/views/reload", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writePlain(w, http.StatusOK, "reload views ok\n")
	}))

	if !plane.HandlesPath("/_golazy/views/reload") {
		t.Fatal("custom control-plane path is not handled")
	}
	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/_golazy/views/reload", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "reload views ok\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestRegisterAddsOwnedEndpointMetadata(t *testing.T) {
	plane := New(Config{})
	err := plane.Register(Endpoint{
		ID:          "postgres.jobs",
		Owner:       "golazy.dev/addons/postgres/jobs",
		Pattern:     "GET /jobs",
		Description: "PostgreSQL job status",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "jobs")
		}),
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/jobs", nil))
	if got, want := response.Body.String(), "jobs"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	var found *EndpointInfo
	for _, endpoint := range plane.Endpoints() {
		if endpoint.ID == "postgres.jobs" {
			copy := endpoint
			found = &copy
			break
		}
	}
	if found == nil {
		t.Fatal("registered endpoint metadata is missing")
	}
	if got, want := found.Owner, "golazy.dev/addons/postgres/jobs"; got != want {
		t.Fatalf("Owner = %q, want %q", got, want)
	}
	if got, want := found.Description, "PostgreSQL job status"; got != want {
		t.Fatalf("Description = %q, want %q", got, want)
	}
	if got, want := found.Method, http.MethodGet; got != want {
		t.Fatalf("Method = %q, want %q", got, want)
	}
}

func TestRegisterPanelPublishesOwnedEndpointMetadata(t *testing.T) {
	plane := New(Config{})
	if err := plane.Register(Endpoint{
		ID:          "postgres.jobs.panel",
		Owner:       "postgres/jobs",
		Pattern:     "/addons/postgres/jobs",
		Description: "PostgreSQL jobs panel handler",
		Handler:     http.NotFoundHandler(),
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := plane.Register(Endpoint{
		ID:          "postgres.jobs.refresh",
		Owner:       "postgres/jobs",
		Pattern:     "POST /addons/postgres/jobs/refresh",
		Description: "Refresh PostgreSQL jobs",
		Handler:     http.NotFoundHandler(),
	}); err != nil {
		t.Fatalf("Register(action) error = %v", err)
	}
	if err := plane.RegisterPanel(Panel{
		ID:          "postgres.jobs",
		Owner:       "postgres/jobs",
		Title:       "PostgreSQL Jobs",
		Description: "Inspect queues and scheduled jobs",
		EndpointID:  "postgres.jobs.panel",
		Actions: []PanelAction{{
			ID:          "refresh",
			Title:       "Refresh",
			Description: "Refresh the jobs snapshot",
			EndpointID:  "postgres.jobs.refresh",
		}},
		Order: 20,
	}); err != nil {
		t.Fatalf("RegisterPanel() error = %v", err)
	}

	panels := plane.Panels()
	if got, want := len(panels), 1; got != want {
		t.Fatalf("panel count = %d, want %d", got, want)
	}
	panel := panels[0]
	if panel.Owner != "postgres/jobs" || panel.Path != "/addons/postgres/jobs" || panel.Method != "ANY" {
		t.Fatalf("panel metadata = %#v", panel)
	}
	if len(panel.Actions) != 1 || panel.Actions[0].ID != "refresh" || panel.Actions[0].Method != http.MethodPost || panel.Actions[0].Path != "/addons/postgres/jobs/refresh" {
		t.Fatalf("panel actions = %#v", panel.Actions)
	}
	panel.Actions[0].Title = "mutated"
	if got := plane.Panels()[0].Actions[0].Title; got != "Refresh" {
		t.Fatalf("Panels returned aliased action metadata: title = %q", got)
	}

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, PanelsPath, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	var discovered PanelsResponse
	if err := json.NewDecoder(response.Body).Decode(&discovered); err != nil {
		t.Fatal(err)
	}
	if discovered.Schema != 1 || len(discovered.Panels) != 1 || discovered.Panels[0].ID != "postgres.jobs" || len(discovered.Panels[0].Actions) != 1 {
		t.Fatalf("discovery response = %#v", discovered)
	}
}

func TestRegisterPanelValidatesOwnedPostOnlyActions(t *testing.T) {
	newPlane := func(t *testing.T) *ControlPlane {
		t.Helper()
		plane := New(Config{})
		for _, endpoint := range []Endpoint{
			{ID: "panel", Owner: "addon", Pattern: "GET /panel", Description: "panel", Handler: http.NotFoundHandler()},
			{ID: "post", Owner: "addon", Pattern: "POST /panel/refresh", Description: "refresh", Handler: http.NotFoundHandler()},
			{ID: "other-post", Owner: "other", Pattern: "POST /other/refresh", Description: "other", Handler: http.NotFoundHandler()},
			{ID: "get", Owner: "addon", Pattern: "GET /panel/action", Description: "get", Handler: http.NotFoundHandler()},
			{ID: "any", Owner: "addon", Pattern: "/panel/any", Description: "any", Handler: http.NotFoundHandler()},
			{ID: "wildcard-post", Owner: "addon", Pattern: "POST /panel/items/{id}", Description: "wildcard", Handler: http.NotFoundHandler()},
			{ID: "prefix-post", Owner: "addon", Pattern: "POST /panel/prefix/", Description: "prefix", Handler: http.NotFoundHandler()},
		} {
			if err := plane.Register(endpoint); err != nil {
				t.Fatal(err)
			}
		}
		return plane
	}
	validAction := PanelAction{ID: "refresh", Title: "Refresh", Description: "Refresh status", EndpointID: "post"}
	validPanel := Panel{ID: "panel", Owner: "addon", Title: "Panel", Description: "Panel description", EndpointID: "panel", Actions: []PanelAction{validAction}}
	if err := newPlane(t).RegisterPanel(validPanel); err != nil {
		t.Fatalf("RegisterPanel() error = %v", err)
	}

	tests := []struct {
		name   string
		change func(*Panel)
		want   string
	}{
		{name: "empty ID", change: func(panel *Panel) { panel.Actions[0].ID = "" }, want: "ID is empty"},
		{name: "empty title", change: func(panel *Panel) { panel.Actions[0].Title = "" }, want: "title is empty"},
		{name: "empty description", change: func(panel *Panel) { panel.Actions[0].Description = "" }, want: "description is empty"},
		{name: "empty endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "" }, want: "endpoint ID is empty"},
		{name: "unknown endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "missing" }, want: "not registered"},
		{name: "different owner", change: func(panel *Panel) { panel.Actions[0].EndpointID = "other-post" }, want: "does not own"},
		{name: "GET endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "get" }, want: "not POST-only"},
		{name: "ANY endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "any" }, want: "not POST-only"},
		{name: "wildcard endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "wildcard-post" }, want: "wildcard pattern"},
		{name: "prefix endpoint", change: func(panel *Panel) { panel.Actions[0].EndpointID = "prefix-post" }, want: "prefix pattern"},
		{name: "duplicate ID", change: func(panel *Panel) { panel.Actions = append(panel.Actions, validAction) }, want: "duplicated"},
		{name: "duplicate method and path", change: func(panel *Panel) {
			copy := validAction
			copy.ID = "again"
			panel.Actions = append(panel.Actions, copy)
		}, want: "duplicates method and path"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			panel := validPanel
			panel.Actions = append([]PanelAction(nil), validPanel.Actions...)
			test.change(&panel)
			err := newPlane(t).RegisterPanel(panel)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("RegisterPanel() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestRegisterPanelValidatesOwnedStableEndpoint(t *testing.T) {
	plane := New(Config{})
	for _, endpoint := range []Endpoint{
		{ID: "owned", Owner: "addon", Pattern: "GET /owned", Description: "owned", Handler: http.NotFoundHandler()},
		{ID: "post-only", Owner: "addon", Pattern: "POST /post-only", Description: "post", Handler: http.NotFoundHandler()},
		{ID: "wildcard", Owner: "addon", Pattern: "GET /items/{id}", Description: "wildcard", Handler: http.NotFoundHandler()},
		{ID: "prefix", Owner: "addon", Pattern: "/prefix/", Description: "prefix", Handler: http.NotFoundHandler()},
	} {
		if err := plane.Register(endpoint); err != nil {
			t.Fatal(err)
		}
	}
	valid := Panel{ID: "panel", Owner: "addon", Title: "Panel", Description: "Panel description", EndpointID: "owned"}
	if err := plane.RegisterPanel(valid); err != nil {
		t.Fatalf("RegisterPanel() error = %v", err)
	}

	tests := []struct {
		name   string
		change func(*Panel)
		want   string
	}{
		{name: "duplicate ID", change: func(panel *Panel) {}, want: "already registered"},
		{name: "unknown endpoint", change: func(panel *Panel) { panel.ID = "unknown"; panel.EndpointID = "missing" }, want: "not registered"},
		{name: "different owner", change: func(panel *Panel) { panel.ID = "other-owner"; panel.Owner = "other" }, want: "does not own"},
		{name: "post only", change: func(panel *Panel) { panel.ID = "post"; panel.EndpointID = "post-only" }, want: "does not support GET"},
		{name: "wildcard", change: func(panel *Panel) { panel.ID = "wildcard"; panel.EndpointID = "wildcard" }, want: "wildcard pattern"},
		{name: "prefix", change: func(panel *Panel) { panel.ID = "prefix"; panel.EndpointID = "prefix" }, want: "prefix pattern"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			panel := valid
			test.change(&panel)
			err := plane.RegisterPanel(panel)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("RegisterPanel() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestRegisterRejectsDuplicateIDAndPattern(t *testing.T) {
	newEndpoint := func(id, pattern string) Endpoint {
		return Endpoint{
			ID:          id,
			Owner:       "test",
			Pattern:     pattern,
			Description: "test endpoint",
			Handler:     http.NotFoundHandler(),
		}
	}

	plane := New(Config{})
	if err := plane.Register(newEndpoint("one", "GET /one")); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	if err := plane.Register(newEndpoint("one", "GET /two")); err == nil || !strings.Contains(err.Error(), "ID") {
		t.Fatalf("duplicate ID error = %v, want endpoint ID error", err)
	}
	if err := plane.Register(newEndpoint("two", "GET /one")); err == nil || !strings.Contains(err.Error(), "pattern") {
		t.Fatalf("duplicate pattern error = %v, want endpoint pattern error", err)
	}
	if err := plane.Register(newEndpoint("wildcard-one", "GET /wildcard/{id}")); err != nil {
		t.Fatalf("wildcard Register() error = %v", err)
	}
	if err := plane.Register(newEndpoint("wildcard-two", "GET /wildcard/{name}")); err == nil || !strings.Contains(err.Error(), "conflicting") {
		t.Fatalf("conflicting pattern error = %v, want ServeMux conflict error", err)
	}
}

func TestRegisterReturnsValidationErrors(t *testing.T) {
	valid := Endpoint{
		ID:          "valid",
		Owner:       "test",
		Pattern:     "GET /valid",
		Description: "valid endpoint",
		Handler:     http.NotFoundHandler(),
	}
	tests := []struct {
		name   string
		change func(*Endpoint)
		want   string
	}{
		{name: "empty ID", change: func(endpoint *Endpoint) { endpoint.ID = "" }, want: "ID is empty"},
		{name: "empty owner", change: func(endpoint *Endpoint) { endpoint.Owner = "" }, want: "owner is empty"},
		{name: "empty description", change: func(endpoint *Endpoint) { endpoint.Description = "" }, want: "description is empty"},
		{name: "empty pattern", change: func(endpoint *Endpoint) { endpoint.Pattern = "" }, want: "pattern is empty"},
		{name: "invalid path", change: func(endpoint *Endpoint) { endpoint.Pattern = "GET valid" }, want: "must start with /"},
		{name: "nil handler", change: func(endpoint *Endpoint) { endpoint.Handler = nil }, want: "handler is nil"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plane := New(Config{})
			endpoint := valid
			test.change(&endpoint)
			err := plane.Register(endpoint)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Register() error = %v, want error containing %q", err, test.want)
			}
		})
	}
}

func TestSealRejectsEndpointAndReadinessRegistration(t *testing.T) {
	plane := New(Config{})
	if err := plane.Register(Endpoint{
		ID:          "panel-endpoint",
		Owner:       "test",
		Pattern:     "GET /panel",
		Description: "panel endpoint",
		Handler:     http.NotFoundHandler(),
	}); err != nil {
		t.Fatal(err)
	}
	plane.Seal()
	plane.Seal()

	if err := plane.Register(Endpoint{
		ID:          "late",
		Owner:       "test",
		Pattern:     "GET /late",
		Description: "late endpoint",
		Handler:     http.NotFoundHandler(),
	}); !errors.Is(err, ErrSealed) {
		t.Fatalf("Register() error = %v, want ErrSealed", err)
	}
	if err := plane.RegisterReadinessCheck(ReadinessCheck{Check: func(context.Context) error { return nil }}); !errors.Is(err, ErrSealed) {
		t.Fatalf("RegisterReadinessCheck() error = %v, want ErrSealed", err)
	}
	if err := plane.RegisterPanel(Panel{
		ID:          "late",
		Owner:       "test",
		Title:       "Late",
		Description: "Late panel",
		EndpointID:  "panel-endpoint",
	}); !errors.Is(err, ErrSealed) {
		t.Fatalf("RegisterPanel() error = %v, want ErrSealed", err)
	}

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("sealed control-plane status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestRegistrationCanContinueWhileEarlyControlPlaneIsServing(t *testing.T) {
	plane := New(Config{})
	server := plane.StandaloneHandler()
	server.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/livez", nil))

	if err := plane.Register(Endpoint{
		ID:          "late",
		Owner:       "test",
		Pattern:     "GET /late",
		Description: "late endpoint",
		Handler:     http.NotFoundHandler(),
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := plane.RegisterReadinessCheck(ReadinessCheck{Check: func(context.Context) error { return nil }}); err != nil {
		t.Fatalf("RegisterReadinessCheck() error = %v", err)
	}
	if err := plane.RegisterPanel(Panel{
		ID:          "late",
		Owner:       "test",
		Title:       "Late",
		Description: "Registered after early serving began",
		EndpointID:  "late",
	}); err != nil {
		t.Fatalf("RegisterPanel() error = %v", err)
	}
}

func TestRegisterReadinessCheckReturnsValidationError(t *testing.T) {
	plane := New(Config{})
	if err := plane.RegisterReadinessCheck(ReadinessCheck{Name: "invalid"}); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("RegisterReadinessCheck() error = %v, want nil check error", err)
	}
	if err := plane.RegisterReadinessCheck(ReadinessCheck{
		Name:  "database",
		Check: func(context.Context) error { return errors.New("offline") },
	}); err != nil {
		t.Fatalf("valid RegisterReadinessCheck() error = %v", err)
	}

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}

func TestConcurrentRegistrationIsSafe(t *testing.T) {
	plane := New(Config{})
	const registrations = 64
	var wait sync.WaitGroup
	errors := make(chan error, registrations*3)
	serveErrors := make(chan error, 1)
	var serveWait sync.WaitGroup
	serveWait.Add(1)
	go func() {
		defer serveWait.Done()
		for index := range registrations * 4 {
			response := httptest.NewRecorder()
			path := "/readyz"
			if index%2 == 1 {
				path = PanelsPath
			}
			plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				serveErrors <- fmt.Errorf("%s status = %d, want %d", path, response.Code, http.StatusOK)
				return
			}
		}
	}()
	for index := range registrations {
		wait.Add(2)
		go func() {
			defer wait.Done()
			endpointID := fmt.Sprintf("endpoint-%d", index)
			err := plane.Register(Endpoint{
				ID:          endpointID,
				Owner:       "test",
				Pattern:     fmt.Sprintf("GET /endpoint/%d", index),
				Description: "concurrent endpoint",
				Handler:     http.NotFoundHandler(),
			})
			errors <- err
			if err == nil {
				errors <- plane.RegisterPanel(Panel{
					ID:          fmt.Sprintf("panel-%d", index),
					Owner:       "test",
					Title:       fmt.Sprintf("Panel %d", index),
					Description: "concurrent panel",
					EndpointID:  endpointID,
				})
			}
		}()
		go func() {
			defer wait.Done()
			errors <- plane.RegisterReadinessCheck(ReadinessCheck{
				Name:  fmt.Sprintf("check-%d", index),
				Check: func(context.Context) error { return nil },
			})
		}()
	}
	wait.Wait()
	serveWait.Wait()
	close(errors)
	close(serveErrors)
	for err := range errors {
		if err != nil {
			t.Fatalf("concurrent registration error = %v", err)
		}
	}
	for err := range serveErrors {
		t.Fatal(err)
	}
	if got, want := len(plane.Endpoints()), registrations+3; got != want {
		t.Fatalf("endpoint count = %d, want %d", got, want)
	}
	if got, want := len(plane.Panels()), registrations; got != want {
		t.Fatalf("panel count = %d, want %d", got, want)
	}

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerMountsControlPlaneBeforeNext(t *testing.T) {
	plane := New(Config{})
	handler := plane.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "app")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if got := response.Body.String(); got != "live\n" {
		t.Fatalf("/livez body = %q, want control plane", got)
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/app", nil))
	if got := response.Body.String(); got != "app" {
		t.Fatalf("/app body = %q, want next handler", got)
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := response.Body.String(); got != "app" {
		t.Fatalf("/ body = %q, want next handler", got)
	}
}

func TestHandlerMountsWildcardEndpointBeforeNext(t *testing.T) {
	plane := New(Config{})
	if err := plane.Register(Endpoint{
		ID:          "jobs.queue",
		Owner:       "postgres/jobs",
		Pattern:     "GET /jobs/{queue}",
		Description: "Queue status",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			_, _ = fmt.Fprint(w, request.PathValue("queue"))
		}),
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	handler := plane.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "app")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/jobs/default", nil))
	if got, want := response.Body.String(), "default"; got != want {
		t.Fatalf("GET body = %q, want %q", got, want)
	}
	if !plane.HandlesPath("/jobs/default") {
		t.Fatal("wildcard endpoint path is not handled")
	}

	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/jobs/default", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestPprofIsExplicit(t *testing.T) {
	plane := New(Config{})
	if plane.HandlesPath("/debug/pprof/") {
		t.Fatal("pprof path is handled by default")
	}

	plane = New(Config{Pprof: true})
	if !plane.HandlesPath("/debug/pprof/") {
		t.Fatal("pprof path is not handled when enabled")
	}
	if !plane.HandlesPath("/debug/pprof/profile") {
		t.Fatal("pprof profile path is not handled when enabled")
	}
}

func TestEnablePprofIsIdempotent(t *testing.T) {
	plane := New(Config{Pprof: true})

	plane.EnablePprof()
	plane.EnablePprof()

	response := httptest.NewRecorder()
	plane.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestStandaloneHandlerServesIndexWithRegisteredEndpoints(t *testing.T) {
	plane := New(Config{
		Metrics: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "metrics")
		}),
		Pprof: true,
	})
	plane.Handle("POST /jobs/run", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "queued")
	}))
	if err := plane.Register(Endpoint{
		ID:          "postgres.jobs.status",
		Owner:       "golazy.dev/addons/postgres/jobs",
		Pattern:     "GET /jobs",
		Description: "PostgreSQL jobs status",
		Handler:     http.NotFoundHandler(),
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	response := httptest.NewRecorder()
	plane.StandaloneHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	body := response.Body.String()
	for _, want := range []string{
		"GoLazy Control Plane",
		"Registered endpoints",
		"GET",
		"/livez",
		"/readyz",
		"/metrics",
		"POST",
		"/jobs/run",
		"postgres.jobs.status",
		"golazy.dev/addons/postgres/jobs",
		"PostgreSQL jobs status",
		"/debug/pprof/",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}
