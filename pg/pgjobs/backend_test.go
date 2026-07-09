package pgjobs

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazymigrate"
	"golazy.dev/pg/pgmigrate"
)

func TestMigrationsIncludeSchedulesAndQueueLimits(t *testing.T) {
	migrations, err := Migrations().LoadMigrations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations {
		if migration.ID == "lazyjobs-202607090001_add_schedules_and_queue_limits" {
			return
		}
	}
	t.Fatalf("schedules migration missing from %#v", migrations)
}

func TestBackendClaimHonorsQueueLimits(t *testing.T) {
	ctx, backend, cleanup := setupBackend(t)
	defer cleanup()

	now := time.Now().UTC()
	if _, err := backend.Insert(ctx, lazyjobs.InsertParams{
		Kind:        "running",
		Queue:       "scraping",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := backend.Claim(ctx, lazyjobs.ClaimParams{Queues: []string{"scraping"}, Now: now}); err != nil || !ok {
		t.Fatalf("claim running scraping = %v %v", ok, err)
	}
	if _, err := backend.Insert(ctx, lazyjobs.InsertParams{
		Kind:        "blocked",
		Queue:       "scraping",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	defaultRecord, err := backend.Insert(ctx, lazyjobs.InsertParams{
		Kind:        "default",
		Queue:       "default",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := backend.Claim(ctx, lazyjobs.ClaimParams{
		Queues:      []string{"default", "scraping"},
		QueueLimits: map[string]int{"scraping": 1},
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("no job claimed")
	}
	if claimed.ID != defaultRecord.ID {
		t.Fatalf("claimed = %#v, want default record", claimed)
	}
}

func TestBackendSchedules(t *testing.T) {
	ctx, backend, cleanup := setupBackend(t)
	defer cleanup()

	now := time.Now().UTC()
	registered, err := backend.RegisterSchedule(ctx, lazyjobs.ScheduleParams{
		Key:       "prices.azure",
		Kind:      "prices.azure",
		Queue:     "scraping",
		Payload:   []byte(`{"source":"azure"}`),
		Interval:  time.Hour,
		NextRunAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if registered.Key != "prices.azure" || registered.Queue != "scraping" {
		t.Fatalf("registered = %#v", registered)
	}
	claimed, ok, err := backend.ClaimSchedule(ctx, lazyjobs.ClaimScheduleParams{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimed.Key != registered.Key {
		t.Fatalf("claimed = %#v ok=%v, want prices.azure", claimed, ok)
	}
	if _, ok, err := backend.ClaimSchedule(ctx, lazyjobs.ClaimScheduleParams{Now: now}); err != nil || ok {
		t.Fatalf("locked claim ok=%v err=%v, want no claim", ok, err)
	}
	nextRunAt := now.Add(time.Hour)
	if err := backend.AdvanceSchedule(ctx, lazyjobs.AdvanceScheduleParams{Key: claimed.Key, NextRunAt: nextRunAt}); err != nil {
		t.Fatal(err)
	}
	schedules, err := backend.ListSchedules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 1 || !schedules[0].NextRunAt.Equal(nextRunAt) {
		t.Fatalf("schedules = %#v, want next run at %s", schedules, nextRunAt)
	}
	if _, err := backend.Insert(ctx, lazyjobs.InsertParams{
		Kind:        "prices.azure",
		Queue:       "scraping",
		ScheduleKey: "prices.azure",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
	}); err != nil {
		t.Fatal(err)
	}
	active, err := backend.HasActiveScheduledJob(ctx, lazyjobs.ActiveScheduledJobParams{ScheduleKey: "prices.azure"})
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("scheduled job is not active")
	}
}

func setupBackend(t *testing.T) (context.Context, *Backend, func()) {
	t.Helper()
	databaseURL := os.Getenv("GOLAZY_PG_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set GOLAZY_PG_DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	reset := func() {
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_job_schedules`)
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_jobs`)
		_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS lazy_migrations`)
	}
	cleanup := func() {
		reset()
		pool.Close()
	}
	reset()

	migrator, err := lazymigrate.New(lazymigrate.Config{
		Backend: pgmigrate.New(pool),
		Sources: []lazymigrate.Source{Migrations()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.Up(ctx, 0); err != nil {
		t.Fatal(err)
	}
	return ctx, New(pool), cleanup
}
