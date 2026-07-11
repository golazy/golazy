package lazyjobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"sync"
	"time"
)

type Config struct {
	Backend      Backend
	Define       func(*JobRunner)
	Workers      int
	PollInterval time.Duration
	Queues       []string
	QueueLimits  map[string]int
}

type JobRunner struct {
	backend      Backend
	scheduler    SchedulerBackend
	pollInterval time.Duration
	workers      int
	queues       []string
	queueLimits  map[string]int

	mu          sync.RWMutex
	definitions map[string]definition
	schedules   map[string]registeredSchedule
	started     bool
	cancel      context.CancelFunc
	done        chan struct{}
}

type definition struct {
	meta Definition
	typ  reflect.Type
}

type registeredSchedule struct {
	key       string
	kind      string
	queue     string
	payload   json.RawMessage
	interval  time.Duration
	nextRunAt time.Time
}

func New(config Config) (*JobRunner, error) {
	if config.Backend == nil {
		return nil, fmt.Errorf("lazyjobs: backend is required")
	}
	workers := config.Workers
	if workers <= 0 {
		workers = 1
	}
	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	configuredQueues := len(config.Queues) > 0
	queues := normalizeQueues(config.Queues)
	runner := &JobRunner{
		backend:      config.Backend,
		pollInterval: pollInterval,
		workers:      workers,
		queues:       queues,
		queueLimits:  normalizeQueueLimits(config.QueueLimits),
		definitions:  map[string]definition{},
		schedules:    map[string]registeredSchedule{},
		done:         closedDone(),
	}
	if config.Define != nil {
		config.Define(runner)
	}
	if !configuredQueues {
		runner.queues = runner.definitionQueues()
	}
	if len(runner.schedules) > 0 {
		scheduler, ok := config.Backend.(SchedulerBackend)
		if !ok {
			return nil, fmt.Errorf("lazyjobs: schedules require backend implementing lazyjobs.SchedulerBackend")
		}
		runner.scheduler = scheduler
		if err := runner.registerSchedules(context.Background()); err != nil {
			return nil, err
		}
	}
	return runner, nil
}

func (r *JobRunner) Register(prototype Job) error {
	if r == nil {
		return fmt.Errorf("lazyjobs: nil runner")
	}
	if prototype == nil {
		return fmt.Errorf("lazyjobs: job prototype is required")
	}
	typ := reflect.TypeOf(prototype)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("lazyjobs: job prototype %T must be a pointer to struct", prototype)
	}
	if reflect.ValueOf(prototype).IsNil() {
		return fmt.Errorf("lazyjobs: job prototype is required")
	}
	kind := prototype.Kind()
	if kind == "" {
		return fmt.Errorf("lazyjobs: job kind is required")
	}
	def := Definition{
		Kind:        kind,
		Type:        typ.Elem().String(),
		Queue:       queueFor(prototype),
		MaxAttempts: maxAttemptsFor(prototype),
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.definitions[kind]; exists {
		return fmt.Errorf("lazyjobs: job kind %q is already registered", kind)
	}
	r.definitions[kind] = definition{meta: def, typ: typ}
	return nil
}

func (r *JobRunner) MustRegister(prototype Job) {
	if err := r.Register(prototype); err != nil {
		panic(err)
	}
}

func (r *JobRunner) Schedule(schedule Schedule) error {
	if r == nil {
		return fmt.Errorf("lazyjobs: nil runner")
	}
	if schedule.Key == "" {
		return fmt.Errorf("lazyjobs: schedule key is required")
	}
	if schedule.Interval <= 0 {
		return fmt.Errorf("lazyjobs: schedule %q interval must be positive", schedule.Key)
	}
	if schedule.Job == nil {
		return fmt.Errorf("lazyjobs: schedule %q job is required", schedule.Key)
	}
	kind := schedule.Job.Kind()
	if kind == "" {
		return fmt.Errorf("lazyjobs: schedule %q job kind is required", schedule.Key)
	}
	if _, ok := r.definition(kind); !ok {
		return fmt.Errorf("lazyjobs: schedule %q job kind %q is not registered", schedule.Key, kind)
	}
	queue := schedule.Queue
	if queue == "" {
		queue = queueFor(schedule.Job)
	} else {
		queue = normalizeQueue(queue)
	}
	payload, err := json.Marshal(schedule.Job)
	if err != nil {
		return fmt.Errorf("lazyjobs: marshal schedule %q: %w", schedule.Key, err)
	}
	nextRunAt := schedule.FirstRunAt
	if nextRunAt.IsZero() {
		nextRunAt = time.Now().UTC()
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.schedules[schedule.Key]; exists {
		return fmt.Errorf("lazyjobs: schedule %q is already registered", schedule.Key)
	}
	r.schedules[schedule.Key] = registeredSchedule{
		key:       schedule.Key,
		kind:      kind,
		queue:     queue,
		payload:   append(json.RawMessage(nil), payload...),
		interval:  schedule.Interval,
		nextRunAt: nextRunAt.UTC(),
	}
	return nil
}

func (r *JobRunner) MustSchedule(schedule Schedule) {
	if err := r.Schedule(schedule); err != nil {
		panic(err)
	}
}

func (r *JobRunner) Enqueue(ctx context.Context, job Job) (Record, error) {
	return r.EnqueueWith(ctx, job)
}

func (r *JobRunner) EnqueueIn(ctx context.Context, job Job, delay time.Duration) (Record, error) {
	return r.EnqueueWith(ctx, job, RunIn(delay))
}

func (r *JobRunner) EnqueueAt(ctx context.Context, job Job, runAt time.Time) (Record, error) {
	return r.EnqueueWith(ctx, job, RunAt(runAt))
}

func (r *JobRunner) EnqueueWith(ctx context.Context, job Job, options ...EnqueueOption) (Record, error) {
	if r == nil || r.backend == nil {
		return Record{}, fmt.Errorf("lazyjobs: runner is not initialized")
	}
	if job == nil {
		return Record{}, fmt.Errorf("lazyjobs: job is required")
	}
	kind := job.Kind()
	if kind == "" {
		return Record{}, fmt.Errorf("lazyjobs: job kind is required")
	}
	if _, ok := r.definition(kind); !ok {
		return Record{}, fmt.Errorf("lazyjobs: job kind %q is not registered", kind)
	}
	enqueueOptions := enqueueOptions{}
	for _, option := range options {
		if option != nil {
			option.applyEnqueueOption(&enqueueOptions)
		}
	}
	payload, err := json.Marshal(job)
	if err != nil {
		return Record{}, fmt.Errorf("lazyjobs: marshal %s: %w", kind, err)
	}
	runAt := enqueueOptions.runAt
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	queue := enqueueOptions.queue
	if queue == "" {
		queue = queueFor(job)
	}
	return r.backend.Insert(ctx, InsertParams{
		Kind:        kind,
		Queue:       queue,
		ScheduleKey: enqueueOptions.scheduleKey,
		Payload:     payload,
		MaxAttempts: maxAttemptsFor(job),
		RunAt:       runAt.UTC(),
	})
}

func (r *JobRunner) Start(ctx context.Context) {
	if r == nil || r.backend == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	r.started = true
	r.cancel = cancel
	r.done = make(chan struct{})
	workers := r.workers
	runScheduler := r.scheduler != nil && len(r.schedules) > 0
	r.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			r.runWorker(runCtx)
		}()
	}
	if runScheduler {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runScheduler(runCtx)
		}()
	}
	go func() {
		wg.Wait()
		close(r.done)
	}()
}

func (r *JobRunner) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	cancel := r.cancel
	done := r.done
	r.started = false
	r.cancel = nil
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		if ctx == nil {
			ctx = context.Background()
		}
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if closer, ok := r.backend.(closeBackend); ok {
		return closer.Close()
	}
	return nil
}

func (r *JobRunner) Snapshot(ctx context.Context) (Snapshot, error) {
	if r == nil || r.backend == nil {
		return Snapshot{}, fmt.Errorf("lazyjobs: runner is not initialized")
	}
	stats, err := r.backend.Stats(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	recent, err := r.backend.List(ctx, ListOptions{Limit: 100})
	if err != nil {
		return Snapshot{}, err
	}
	definitions := r.Definitions()
	schedules, err := r.ScheduleDefinitions(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Running:     r.Running(),
		Definitions: definitions,
		Schedules:   schedules,
		QueueLimits: r.QueueLimitStates(stats),
		Stats:       stats,
		Recent:      recent,
	}, nil
}

func (r *JobRunner) Definitions() []Definition {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	definitions := make([]Definition, 0, len(r.definitions))
	for _, def := range r.definitions {
		definitions = append(definitions, def.meta)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Kind < definitions[j].Kind
	})
	return definitions
}

func (r *JobRunner) ScheduleDefinitions(ctx context.Context) ([]ScheduleDefinition, error) {
	if r == nil {
		return nil, nil
	}
	r.mu.RLock()
	hasSchedules := len(r.schedules) > 0
	r.mu.RUnlock()
	if !hasSchedules {
		return nil, nil
	}
	if r.scheduler == nil {
		return nil, nil
	}
	records, err := r.scheduler.ListSchedules(ctx)
	if err != nil {
		return nil, err
	}
	definitions := make([]ScheduleDefinition, 0, len(records))
	for _, record := range records {
		definitions = append(definitions, ScheduleDefinition{
			Key:       record.Key,
			Kind:      record.Kind,
			Queue:     record.Queue,
			Interval:  record.Interval.String(),
			NextRunAt: record.NextRunAt,
		})
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Key < definitions[j].Key
	})
	return definitions, nil
}

func (r *JobRunner) QueueLimitStates(stats Stats) []QueueLimitState {
	if r == nil || len(r.queueLimits) == 0 {
		return nil
	}
	states := make([]QueueLimitState, 0, len(r.queueLimits))
	for queue, maxRunning := range r.queueLimits {
		running := 0
		if stats.ByQueueState != nil && stats.ByQueueState[queue] != nil {
			running = stats.ByQueueState[queue][StateRunning]
		}
		available := maxRunning - running
		if available < 0 {
			available = 0
		}
		states = append(states, QueueLimitState{
			Queue:      queue,
			MaxRunning: maxRunning,
			Running:    running,
			Available:  available,
		})
	}
	sort.Slice(states, func(i, j int) bool {
		return states[i].Queue < states[j].Queue
	})
	return states
}

func (r *JobRunner) Running() bool {
	if r == nil {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.started
}

func (r *JobRunner) runWorker(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		claimed, ok, err := r.backend.Claim(ctx, ClaimParams{Queues: r.queues, QueueLimits: r.queueLimits, Now: time.Now().UTC()})
		if err == nil && ok {
			r.work(ctx, claimed)
			timer.Reset(0)
			continue
		}
		timer.Reset(r.pollInterval)
	}
}

func (r *JobRunner) work(ctx context.Context, record Record) {
	job, err := r.decode(record)
	if err != nil {
		_ = r.backend.Discard(ctx, DiscardParams{ID: record.ID, LastError: err.Error()})
		return
	}
	err = runJob(ctx, job)
	if err == nil {
		_ = r.backend.Complete(ctx, record.ID)
		return
	}
	if record.Attempt >= record.MaxAttempts {
		_ = r.backend.Discard(ctx, DiscardParams{ID: record.ID, LastError: err.Error()})
		return
	}
	delay := retryDelayFor(job, record.Attempt, err)
	_ = r.backend.Retry(ctx, RetryParams{
		ID:        record.ID,
		RunAt:     time.Now().UTC().Add(delay),
		LastError: err.Error(),
	})
}

func (r *JobRunner) registerSchedules(ctx context.Context) error {
	r.mu.RLock()
	schedules := make([]registeredSchedule, 0, len(r.schedules))
	for _, schedule := range r.schedules {
		schedules = append(schedules, schedule)
	}
	r.mu.RUnlock()
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].key < schedules[j].key
	})
	for _, schedule := range schedules {
		if _, err := r.scheduler.RegisterSchedule(ctx, ScheduleParams{
			Key:       schedule.key,
			Kind:      schedule.kind,
			Queue:     schedule.queue,
			Payload:   append(json.RawMessage(nil), schedule.payload...),
			Interval:  schedule.interval,
			NextRunAt: schedule.nextRunAt,
		}); err != nil {
			return fmt.Errorf("lazyjobs: register schedule %q: %w", schedule.key, err)
		}
	}
	return nil
}

func (r *JobRunner) runScheduler(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		claimed, ok, err := r.scheduler.ClaimSchedule(ctx, ClaimScheduleParams{Now: time.Now().UTC()})
		if err == nil && ok {
			r.scheduleClaimed(ctx, claimed)
			timer.Reset(0)
			continue
		}
		timer.Reset(r.pollInterval)
	}
}

func (r *JobRunner) scheduleClaimed(ctx context.Context, schedule ScheduleRecord) {
	now := time.Now().UTC()
	nextRunAt := nextScheduledRun(schedule.NextRunAt, schedule.Interval, now)
	active, err := r.scheduler.HasActiveScheduledJob(ctx, ActiveScheduledJobParams{ScheduleKey: schedule.Key})
	if err == nil && !active {
		_, err = r.backend.Insert(ctx, InsertParams{
			Kind:        schedule.Kind,
			Queue:       schedule.Queue,
			ScheduleKey: schedule.Key,
			Payload:     append(json.RawMessage(nil), schedule.Payload...),
			MaxAttempts: maxAttemptsForScheduled(r, schedule.Kind),
			RunAt:       now,
		})
	}
	if err != nil {
		nextRunAt = now.Add(r.pollInterval)
	}
	_ = r.scheduler.AdvanceSchedule(ctx, AdvanceScheduleParams{Key: schedule.Key, NextRunAt: nextRunAt})
}

func maxAttemptsForScheduled(r *JobRunner, kind string) int {
	def, ok := r.definition(kind)
	if !ok {
		return normalizeAttempts(0)
	}
	job := reflect.New(def.typ.Elem()).Interface()
	if typed, ok := job.(Job); ok {
		return maxAttemptsFor(typed)
	}
	return normalizeAttempts(0)
}

func nextScheduledRun(previous time.Time, interval time.Duration, now time.Time) time.Time {
	if interval <= 0 {
		return now
	}
	if previous.IsZero() {
		return now.Add(interval)
	}
	next := previous.Add(interval)
	for !next.After(now) {
		next = next.Add(interval)
	}
	return next.UTC()
}

func (r *JobRunner) decode(record Record) (Job, error) {
	def, ok := r.definition(record.Kind)
	if !ok {
		return nil, fmt.Errorf("lazyjobs: job kind %q is not registered", record.Kind)
	}
	value := reflect.New(def.typ.Elem()).Interface()
	if err := json.Unmarshal(record.Payload, value); err != nil {
		return nil, fmt.Errorf("lazyjobs: decode %s: %w", record.Kind, err)
	}
	job, ok := value.(Job)
	if !ok {
		return nil, fmt.Errorf("lazyjobs: %s no longer implements Job", def.typ)
	}
	return job, nil
}

func (r *JobRunner) definition(kind string) (definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.definitions[kind]
	return def, ok
}

func (r *JobRunner) definitionQueues() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.definitions) == 0 && len(r.schedules) == 0 && len(r.queueLimits) == 0 {
		return []string{DefaultQueue}
	}
	queues := make(map[string]bool, len(r.definitions)+len(r.schedules)+len(r.queueLimits))
	for _, def := range r.definitions {
		queues[def.meta.Queue] = true
	}
	for _, schedule := range r.schedules {
		queues[schedule.queue] = true
	}
	for queue := range r.queueLimits {
		queues[queue] = true
	}
	out := make([]string, 0, len(queues))
	for queue := range queues {
		out = append(out, queue)
	}
	sort.Strings(out)
	return out
}

func runJob(ctx context.Context, job Job) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v\n%s", recovered, debug.Stack())
		}
	}()
	return job.Work(ctx)
}

func queueFor(job Job) string {
	if named, ok := job.(QueueNamer); ok {
		return normalizeQueue(named.JobQueue())
	}
	return DefaultQueue
}

func normalizeQueues(queues []string) []string {
	if len(queues) == 0 {
		return []string{DefaultQueue}
	}
	out := append([]string(nil), queues...)
	for index, queue := range out {
		out[index] = normalizeQueue(queue)
	}
	return out
}

func normalizeQueueLimits(limits map[string]int) map[string]int {
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
	if len(out) == 0 {
		return nil
	}
	return out
}

func maxAttemptsFor(job Job) int {
	if policy, ok := job.(RetryPolicy); ok {
		return normalizeAttempts(policy.JobMaxAttempts())
	}
	return normalizeAttempts(0)
}

func retryDelayFor(job Job, attempt int, err error) time.Duration {
	if policy, ok := job.(RetryPolicy); ok {
		delay := policy.JobRetryDelay(attempt, err)
		if delay > 0 {
			return delay
		}
	}
	return BaseJob{}.JobRetryDelay(attempt, err)
}

func closedDone() chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

var ErrNoWork = errors.New("lazyjobs: no work")
