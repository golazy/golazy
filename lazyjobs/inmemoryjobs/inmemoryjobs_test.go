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
