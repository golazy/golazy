package inmemoryjobs

import (
	"context"
	"sort"
	"sync"
	"time"

	"golazy.dev/lazyjobs"
)

type Backend struct {
	mu     sync.Mutex
	nextID int64
	jobs   map[int64]lazyjobs.Record
}

func New() *Backend {
	return &Backend{nextID: 1, jobs: map[int64]lazyjobs.Record{}}
}

func (b *Backend) Insert(_ context.Context, params lazyjobs.InsertParams) (lazyjobs.Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().UTC()
	runAt := params.RunAt
	if runAt.IsZero() {
		runAt = now
	}
	record := lazyjobs.Record{
		ID:          b.nextID,
		Kind:        params.Kind,
		Queue:       normalizeQueue(params.Queue),
		Payload:     append([]byte(nil), params.Payload...),
		State:       lazyjobs.StatePending,
		MaxAttempts: normalizeAttempts(params.MaxAttempts),
		RunAt:       runAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	b.nextID++
	b.jobs[record.ID] = record
	return clone(record), nil
}

func (b *Backend) Claim(_ context.Context, params lazyjobs.ClaimParams) (lazyjobs.Record, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	queues := queueSet(params.Queues)
	var selected *lazyjobs.Record
	for _, record := range b.jobs {
		if !claimable(record.State) || record.RunAt.After(now) || !queues[record.Queue] {
			continue
		}
		candidate := record
		if selected == nil || candidate.RunAt.Before(selected.RunAt) || candidate.RunAt.Equal(selected.RunAt) && candidate.ID < selected.ID {
			selected = &candidate
		}
	}
	if selected == nil {
		return lazyjobs.Record{}, false, nil
	}
	selected.State = lazyjobs.StateRunning
	selected.Attempt++
	selected.UpdatedAt = now
	b.jobs[selected.ID] = *selected
	return clone(*selected), true, nil
}

func (b *Backend) Complete(_ context.Context, id int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	record := b.jobs[id]
	record.State = lazyjobs.StateSucceeded
	record.UpdatedAt = time.Now().UTC()
	b.jobs[id] = record
	return nil
}

func (b *Backend) Retry(_ context.Context, params lazyjobs.RetryParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	record := b.jobs[params.ID]
	record.State = lazyjobs.StateRetrying
	record.RunAt = params.RunAt
	record.LastError = params.LastError
	record.UpdatedAt = time.Now().UTC()
	b.jobs[params.ID] = record
	return nil
}

func (b *Backend) Discard(_ context.Context, params lazyjobs.DiscardParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	record := b.jobs[params.ID]
	record.State = lazyjobs.StateDiscarded
	record.LastError = params.LastError
	record.UpdatedAt = time.Now().UTC()
	b.jobs[params.ID] = record
	return nil
}

func (b *Backend) List(_ context.Context, options lazyjobs.ListOptions) ([]lazyjobs.Record, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	records := make([]lazyjobs.Record, 0, len(b.jobs))
	for _, record := range b.jobs {
		records = append(records, clone(record))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].UpdatedAt.Equal(records[j].UpdatedAt) {
			return records[i].ID > records[j].ID
		}
		return records[i].UpdatedAt.After(records[j].UpdatedAt)
	})
	if options.Limit > 0 && len(records) > options.Limit {
		records = records[:options.Limit]
	}
	return records, nil
}

func (b *Backend) Stats(_ context.Context) (lazyjobs.Stats, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	stats := lazyjobs.Stats{
		ByState: map[lazyjobs.State]int{},
		ByKind:  map[string]int{},
		ByQueue: map[string]int{},
	}
	for _, record := range b.jobs {
		stats.Total++
		stats.ByState[record.State]++
		stats.ByKind[record.Kind]++
		stats.ByQueue[record.Queue]++
	}
	return stats, nil
}

func claimable(state lazyjobs.State) bool {
	return state == lazyjobs.StatePending || state == lazyjobs.StateRetrying
}

func queueSet(queues []string) map[string]bool {
	if len(queues) == 0 {
		return map[string]bool{lazyjobs.DefaultQueue: true}
	}
	out := make(map[string]bool, len(queues))
	for _, queue := range queues {
		out[normalizeQueue(queue)] = true
	}
	return out
}

func normalizeQueue(queue string) string {
	if queue == "" {
		return lazyjobs.DefaultQueue
	}
	return queue
}

func normalizeAttempts(attempts int) int {
	if attempts <= 0 {
		return 25
	}
	return attempts
}

func clone(record lazyjobs.Record) lazyjobs.Record {
	record.Payload = append([]byte(nil), record.Payload...)
	return record
}
