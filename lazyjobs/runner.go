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
}

type JobRunner struct {
	backend      Backend
	pollInterval time.Duration
	workers      int
	queues       []string

	mu          sync.RWMutex
	definitions map[string]definition
	started     bool
	cancel      context.CancelFunc
	done        chan struct{}
}

type definition struct {
	meta Definition
	typ  reflect.Type
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
		definitions:  map[string]definition{},
		done:         closedDone(),
	}
	if config.Define != nil {
		config.Define(runner)
	}
	if !configuredQueues {
		runner.queues = runner.definitionQueues()
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

func (r *JobRunner) Enqueue(ctx context.Context, job Job) (Record, error) {
	return r.EnqueueAt(ctx, job, time.Now().UTC())
}

func (r *JobRunner) EnqueueIn(ctx context.Context, job Job, delay time.Duration) (Record, error) {
	if delay < 0 {
		delay = 0
	}
	return r.EnqueueAt(ctx, job, time.Now().UTC().Add(delay))
}

func (r *JobRunner) EnqueueAt(ctx context.Context, job Job, runAt time.Time) (Record, error) {
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
	payload, err := json.Marshal(job)
	if err != nil {
		return Record{}, fmt.Errorf("lazyjobs: marshal %s: %w", kind, err)
	}
	if runAt.IsZero() {
		runAt = time.Now().UTC()
	}
	return r.backend.Insert(ctx, InsertParams{
		Kind:        kind,
		Queue:       queueFor(job),
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
	r.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(workers)
	for workerID := 0; workerID < workers; workerID++ {
		go func() {
			defer wg.Done()
			r.runWorker(runCtx)
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
	return Snapshot{
		Running:     r.Running(),
		Definitions: definitions,
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
		claimed, ok, err := r.backend.Claim(ctx, ClaimParams{Queues: r.queues, Now: time.Now().UTC()})
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
	if len(r.definitions) == 0 {
		return []string{DefaultQueue}
	}
	queues := make(map[string]bool, len(r.definitions))
	for _, def := range r.definitions {
		queues[def.meta.Queue] = true
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var ErrNoWork = errors.New("lazyjobs: no work")
