package inmemoryjobs_test

import (
	"context"
	"testing"
	"time"

	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
)

func TestBackendClaimOrderAndStats(t *testing.T) {
	backend := inmemoryjobs.New()
	now := time.Now().UTC()
	if _, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "later",
		Queue:       "default",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	early, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "early",
		Queue:       "default",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := backend.Claim(context.Background(), lazyjobs.ClaimParams{Queues: []string{"default"}, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("no job claimed")
	}
	if claimed.ID != early.ID || claimed.Attempt != 1 || claimed.State != lazyjobs.StateRunning {
		t.Fatalf("claimed = %#v, want early running attempt 1", claimed)
	}

	stats, err := backend.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Total != 2 || stats.ByState[lazyjobs.StateRunning] != 1 || stats.ByState[lazyjobs.StatePending] != 1 {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestBackendClaimHonorsQueueLimits(t *testing.T) {
	backend := inmemoryjobs.New()
	now := time.Now().UTC()
	if _, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "running",
		Queue:       "scraping",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := backend.Claim(context.Background(), lazyjobs.ClaimParams{Queues: []string{"scraping"}, Now: now}); err != nil || !ok {
		t.Fatalf("claim running scraping = %v %v", ok, err)
	}
	if _, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "blocked",
		Queue:       "scraping",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	}); err != nil {
		t.Fatal(err)
	}
	defaultRecord, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "default",
		Queue:       "default",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
		RunAt:       now,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := backend.Claim(context.Background(), lazyjobs.ClaimParams{
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
	backend := inmemoryjobs.New()
	now := time.Now().UTC()
	registered, err := backend.RegisterSchedule(context.Background(), lazyjobs.ScheduleParams{
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

	claimed, ok, err := backend.ClaimSchedule(context.Background(), lazyjobs.ClaimScheduleParams{Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || claimed.Key != registered.Key {
		t.Fatalf("claimed = %#v ok=%v, want prices.azure", claimed, ok)
	}
	if _, ok, err := backend.ClaimSchedule(context.Background(), lazyjobs.ClaimScheduleParams{Now: now}); err != nil || ok {
		t.Fatalf("locked claim ok=%v err=%v, want no claim", ok, err)
	}
	nextRunAt := now.Add(time.Hour)
	if err := backend.AdvanceSchedule(context.Background(), lazyjobs.AdvanceScheduleParams{Key: claimed.Key, NextRunAt: nextRunAt}); err != nil {
		t.Fatal(err)
	}
	schedules, err := backend.ListSchedules(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(schedules) != 1 || !schedules[0].NextRunAt.Equal(nextRunAt) {
		t.Fatalf("schedules = %#v, want next run at %s", schedules, nextRunAt)
	}
	if _, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "prices.azure",
		Queue:       "scraping",
		ScheduleKey: "prices.azure",
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
	}); err != nil {
		t.Fatal(err)
	}
	active, err := backend.HasActiveScheduledJob(context.Background(), lazyjobs.ActiveScheduledJobParams{ScheduleKey: "prices.azure"})
	if err != nil {
		t.Fatal(err)
	}
	if !active {
		t.Fatal("scheduled job is not active")
	}
}
