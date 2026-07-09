package inmemoryjobs

import (
	"context"
	"sort"
	"sync"
	"time"

	"golazy.dev/lazyjobs"
)

type Backend struct {
	mu        sync.Mutex
	nextID    int64
	jobs      map[int64]lazyjobs.Record
	schedules map[string]scheduleRecord
}

func New() *Backend {
	return &Backend{nextID: 1, jobs: map[int64]lazyjobs.Record{}, schedules: map[string]scheduleRecord{}}
}

type scheduleRecord struct {
	record   lazyjobs.ScheduleRecord
	lockedAt time.Time
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
		ScheduleKey: params.ScheduleKey,
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
	queueLimits := normalizedQueueLimits(params.QueueLimits)
	running := b.runningByQueue()
	var selected *lazyjobs.Record
	for _, record := range b.jobs {
		if !claimable(record.State) || record.RunAt.After(now) || !queues[record.Queue] {
			continue
		}
		if limit, ok := queueLimits[record.Queue]; ok && running[record.Queue] >= limit {
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
		ByState:      map[lazyjobs.State]int{},
		ByKind:       map[string]int{},
		ByQueue:      map[string]int{},
		ByQueueState: map[string]map[lazyjobs.State]int{},
	}
	for _, record := range b.jobs {
		stats.Total++
		stats.ByState[record.State]++
		stats.ByKind[record.Kind]++
		stats.ByQueue[record.Queue]++
		if stats.ByQueueState[record.Queue] == nil {
			stats.ByQueueState[record.Queue] = map[lazyjobs.State]int{}
		}
		stats.ByQueueState[record.Queue][record.State]++
	}
	return stats, nil
}

func (b *Backend) RegisterSchedule(_ context.Context, params lazyjobs.ScheduleParams) (lazyjobs.ScheduleRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now().UTC()
	nextRunAt := params.NextRunAt
	if nextRunAt.IsZero() {
		nextRunAt = now
	}
	record := lazyjobs.ScheduleRecord{
		Key:       params.Key,
		Kind:      params.Kind,
		Queue:     normalizeQueue(params.Queue),
		Payload:   append([]byte(nil), params.Payload...),
		Interval:  params.Interval,
		NextRunAt: nextRunAt.UTC(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if existing, ok := b.schedules[params.Key]; ok {
		record.NextRunAt = existing.record.NextRunAt
		record.CreatedAt = existing.record.CreatedAt
	}
	b.schedules[params.Key] = scheduleRecord{record: record}
	return cloneSchedule(record), nil
}

func (b *Backend) ClaimSchedule(_ context.Context, params lazyjobs.ClaimScheduleParams) (lazyjobs.ScheduleRecord, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := params.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	staleLock := now.Add(-5 * time.Minute)
	var selected *scheduleRecord
	for _, record := range b.schedules {
		if record.record.NextRunAt.After(now) {
			continue
		}
		if !record.lockedAt.IsZero() && record.lockedAt.After(staleLock) {
			continue
		}
		candidate := record
		if selected == nil || candidate.record.NextRunAt.Before(selected.record.NextRunAt) || candidate.record.NextRunAt.Equal(selected.record.NextRunAt) && candidate.record.Key < selected.record.Key {
			selected = &candidate
		}
	}
	if selected == nil {
		return lazyjobs.ScheduleRecord{}, false, nil
	}
	selected.lockedAt = now
	selected.record.UpdatedAt = now
	b.schedules[selected.record.Key] = *selected
	return cloneSchedule(selected.record), true, nil
}

func (b *Backend) AdvanceSchedule(_ context.Context, params lazyjobs.AdvanceScheduleParams) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	record, ok := b.schedules[params.Key]
	if !ok {
		return nil
	}
	record.record.NextRunAt = params.NextRunAt.UTC()
	record.record.UpdatedAt = time.Now().UTC()
	record.lockedAt = time.Time{}
	b.schedules[params.Key] = record
	return nil
}

func (b *Backend) HasActiveScheduledJob(_ context.Context, params lazyjobs.ActiveScheduledJobParams) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, record := range b.jobs {
		if record.ScheduleKey == params.ScheduleKey && activeScheduledJob(record.State) {
			return true, nil
		}
	}
	return false, nil
}

func (b *Backend) ListSchedules(_ context.Context) ([]lazyjobs.ScheduleRecord, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	records := make([]lazyjobs.ScheduleRecord, 0, len(b.schedules))
	for _, record := range b.schedules {
		records = append(records, cloneSchedule(record.record))
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})
	return records, nil
}

func claimable(state lazyjobs.State) bool {
	return state == lazyjobs.StatePending || state == lazyjobs.StateRetrying
}

func activeScheduledJob(state lazyjobs.State) bool {
	return state == lazyjobs.StatePending || state == lazyjobs.StateRetrying || state == lazyjobs.StateRunning
}

func (b *Backend) runningByQueue() map[string]int {
	out := map[string]int{}
	for _, record := range b.jobs {
		if record.State == lazyjobs.StateRunning {
			out[record.Queue]++
		}
	}
	return out
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

func normalizedQueueLimits(limits map[string]int) map[string]int {
	if len(limits) == 0 {
		return nil
	}
	out := make(map[string]int, len(limits))
	for queue, limit := range limits {
		if limit <= 0 {
			continue
		}
		out[normalizeQueue(queue)] = limit
	}
	return out
}

func clone(record lazyjobs.Record) lazyjobs.Record {
	record.Payload = append([]byte(nil), record.Payload...)
	return record
}

func cloneSchedule(record lazyjobs.ScheduleRecord) lazyjobs.ScheduleRecord {
	record.Payload = append([]byte(nil), record.Payload...)
	return record
}
