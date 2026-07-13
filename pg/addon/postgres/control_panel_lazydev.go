//go:build lazydev

package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyaddon"
	"golazy.dev/lazyapp"
	"golazy.dev/lazycontrolplane"
)

const (
	lazyDevControlPlaneCallbackID = "postgres/control-plane"
	lazyDevEndpointID             = "postgres.pool-status"
	lazyDevPingEndpointID         = "postgres.pool-ping"
	lazyDevPanelID                = "postgres"
	lazyDevPath                   = "/addons/postgres"
	lazyDevPingPath               = "/addons/postgres/ping"
	lazyDevPingTimeout            = 2 * time.Second
)

var pingPostgresPool = func(ctx context.Context, pool *pgxpool.Pool) error {
	return pool.Ping(ctx)
}

type lazyDevPoolResponse struct {
	Healthy bool                  `json:"healthy"`
	Status  string                `json:"status"`
	PingMS  float64               `json:"ping_ms"`
	Pool    lazyDevPoolStatistics `json:"pool"`
}

type lazyDevPoolStatistics struct {
	MaxConnections          int32   `json:"max_connections"`
	TotalConnections        int32   `json:"total_connections"`
	AcquiredConnections     int32   `json:"acquired_connections"`
	IdleConnections         int32   `json:"idle_connections"`
	ConstructingConnections int32   `json:"constructing_connections"`
	AcquireCount            int64   `json:"acquire_count"`
	AcquireDurationMS       float64 `json:"acquire_duration_ms"`
	CanceledAcquireCount    int64   `json:"canceled_acquire_count"`
	EmptyAcquireCount       int64   `json:"empty_acquire_count"`
	EmptyAcquireWaitMS      float64 `json:"empty_acquire_wait_ms"`
	NewConnectionsCount     int64   `json:"new_connections_count"`
	MaxIdleDestroyCount     int64   `json:"max_idle_destroy_count"`
	MaxLifetimeDestroyCount int64   `json:"max_lifetime_destroy_count"`
}

func init() {
	lazyaddon.MustOn(addonRegistration, lazyapp.ControlPlaneHook, lazyaddon.CallbackOptions{ID: lazyDevControlPlaneCallbackID}, registerLazyDevPanel)
}

func registerLazyDevPanel(event *lazyapp.ControlPlaneEvent) error {
	if event == nil || event.ControlPlane == nil {
		return fmt.Errorf("postgres add-on: control-plane registrar is required")
	}
	pool, err := lazyaddon.Require(event.Addons, PoolCapability)
	if err != nil {
		return fmt.Errorf("postgres add-on: control-plane pool: %w", err)
	}
	if err := event.ControlPlane.Register(lazycontrolplane.Endpoint{
		ID:          lazyDevEndpointID,
		Owner:       AddonID,
		Pattern:     "GET " + lazyDevPath,
		Description: "PostgreSQL pool health and connection statistics",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeLazyDevPoolResponse(w, r, pool)
		}),
	}); err != nil {
		return fmt.Errorf("postgres add-on: register control-plane endpoint: %w", err)
	}
	if err := event.ControlPlane.Register(lazycontrolplane.Endpoint{
		ID:          lazyDevPingEndpointID,
		Owner:       AddonID,
		Pattern:     "POST " + lazyDevPingPath,
		Description: "Ping the PostgreSQL pool and refresh its statistics",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeLazyDevPoolResponse(w, r, pool)
		}),
	}); err != nil {
		return fmt.Errorf("postgres add-on: register control-plane ping endpoint: %w", err)
	}
	if err := event.ControlPlane.RegisterPanel(lazycontrolplane.Panel{
		ID:          lazyDevPanelID,
		Owner:       AddonID,
		Title:       "PostgreSQL",
		Description: "Pool health and connection usage",
		EndpointID:  lazyDevEndpointID,
		Actions: []lazycontrolplane.PanelAction{{
			ID:          "ping",
			Title:       "Ping database",
			Description: "Run a fresh database ping and show the updated pool status",
			EndpointID:  lazyDevPingEndpointID,
		}},
		Order: 300,
	}); err != nil {
		return fmt.Errorf("postgres add-on: register developer panel: %w", err)
	}
	return nil
}

func writeLazyDevPoolResponse(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), lazyDevPingTimeout)
	defer cancel()
	err := pingPostgresPool(ctx, pool)
	response := lazyDevPoolResponse{
		Healthy: err == nil,
		Status:  "available",
		PingMS:  durationMilliseconds(time.Since(started)),
		Pool:    lazyDevStatistics(pool.Stat()),
	}
	if err != nil {
		response.Status = "unavailable"
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "postgres: encode pool status\n", http.StatusInternalServerError)
	}
}

func lazyDevStatistics(stat *pgxpool.Stat) lazyDevPoolStatistics {
	return lazyDevPoolStatistics{
		MaxConnections:          stat.MaxConns(),
		TotalConnections:        stat.TotalConns(),
		AcquiredConnections:     stat.AcquiredConns(),
		IdleConnections:         stat.IdleConns(),
		ConstructingConnections: stat.ConstructingConns(),
		AcquireCount:            stat.AcquireCount(),
		AcquireDurationMS:       durationMilliseconds(stat.AcquireDuration()),
		CanceledAcquireCount:    stat.CanceledAcquireCount(),
		EmptyAcquireCount:       stat.EmptyAcquireCount(),
		EmptyAcquireWaitMS:      durationMilliseconds(stat.EmptyAcquireWaitTime()),
		NewConnectionsCount:     stat.NewConnsCount(),
		MaxIdleDestroyCount:     stat.MaxIdleDestroyCount(),
		MaxLifetimeDestroyCount: stat.MaxLifetimeDestroyCount(),
	}
}

func durationMilliseconds(duration time.Duration) float64 {
	return float64(duration) / float64(time.Millisecond)
}
