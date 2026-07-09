package lazyjobs_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazyjobs"
	"golazy.dev/lazyjobs/inmemoryjobs"
)

type testJob struct {
	lazyjobs.BaseJob
	Value string `json:"value"`
}

func (j *testJob) Kind() string { return "test.job" }

func (j *testJob) Work(ctx context.Context) error {
	if testWorked != nil {
		testWorked <- j.Value
	}
	return nil
}

var testWorked chan string

func TestRunnerWorksJob(t *testing.T) {
	testWorked = make(chan string, 1)
	defer func() { testWorked = nil }()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend: inmemoryjobs.New(),
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&testJob{})
		},
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.Enqueue(context.Background(), &testJob{Value: "hello"}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())

	select {
	case got := <-testWorked:
		if got != "hello" {
			t.Fatalf("worked value = %q, want hello", got)
		}
	case <-time.After(time.Second):
		t.Fatal("job did not run")
	}
}

func TestRunnerRejectsUnregisteredEnqueue(t *testing.T) {
	runner, err := lazyjobs.New(lazyjobs.Config{Backend: inmemoryjobs.New()})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.Enqueue(context.Background(), &testJob{Value: "hello"}); err == nil {
		t.Fatal("enqueue succeeded for unregistered job")
	}
}

func TestRunnerClaimsRegisteredCustomQueueByDefault(t *testing.T) {
	testWorked = make(chan string, 1)
	defer func() { testWorked = nil }()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend: inmemoryjobs.New(),
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&customQueueJob{})
		},
		PollInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.Enqueue(context.Background(), &customQueueJob{testJob: testJob{Value: "custom"}}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())

	select {
	case got := <-testWorked:
		if got != "custom" {
			t.Fatalf("worked value = %q, want custom", got)
		}
	case <-time.After(time.Second):
		t.Fatal("custom queue job did not run")
	}
}

func TestRunnerEnqueueWithOverridesQueue(t *testing.T) {
	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend: backend,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&testJob{})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.EnqueueWith(context.Background(), &testJob{Value: "hello"}, lazyjobs.Queue("scraping")); err != nil {
		t.Fatal(err)
	}
	records, err := backend.List(context.Background(), lazyjobs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Queue != "scraping" {
		t.Fatalf("records = %#v, want one scraping record", records)
	}
}

func TestRunnerEnqueueInKeepsDelayedCompatibility(t *testing.T) {
	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend: backend,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&testJob{})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	before := time.Now().UTC()
	if _, err := runner.EnqueueIn(context.Background(), &testJob{Value: "later"}, time.Minute); err != nil {
		t.Fatal(err)
	}
	records, err := backend.List(context.Background(), lazyjobs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].RunAt.Before(before.Add(59*time.Second)) {
		t.Fatalf("records = %#v, want delayed run_at", records)
	}
}

func TestRunnerHonorsGlobalWorkerLimit(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	blockingWorked = blockingJobState{started: started, release: release}
	defer func() { blockingWorked = blockingJobState{} }()

	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend:      inmemoryjobs.New(),
		Workers:      1,
		PollInterval: time.Millisecond,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&blockingJob{})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())
	defer close(release)

	if _, err := runner.Enqueue(context.Background(), &blockingJob{Name: "first"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Enqueue(context.Background(), &blockingJob{Name: "second"}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())

	if got := waitStarted(t, started); got != "first" {
		t.Fatalf("first started = %q", got)
	}
	assertNoStart(t, started, 25*time.Millisecond)
	release <- struct{}{}
	if got := waitStarted(t, started); got != "second" {
		t.Fatalf("second started = %q", got)
	}
}

func TestRunnerHonorsPerQueueLimit(t *testing.T) {
	started := make(chan string, 3)
	release := make(chan struct{})
	blockingWorked = blockingJobState{started: started, release: release}
	defer func() { blockingWorked = blockingJobState{} }()

	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend:      inmemoryjobs.New(),
		Workers:      3,
		PollInterval: time.Millisecond,
		Queues:       []string{"default", "scraping"},
		QueueLimits:  map[string]int{"scraping": 1},
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&blockingJob{})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())
	defer close(release)

	if _, err := runner.Enqueue(context.Background(), &blockingJob{Name: "scrape-1", QueueName: "scraping"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Enqueue(context.Background(), &blockingJob{Name: "scrape-2", QueueName: "scraping"}); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Enqueue(context.Background(), &blockingJob{Name: "normal"}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())

	first := waitStarted(t, started)
	second := waitStarted(t, started)
	startedSet := map[string]bool{first: true, second: true}
	if !startedSet["scrape-1"] || !startedSet["normal"] {
		t.Fatalf("started = %v, want scrape-1 and normal before scrape-2", startedSet)
	}
	assertNoStart(t, started, 25*time.Millisecond)
	release <- struct{}{}
	release <- struct{}{}
	if got := waitStarted(t, started); got != "scrape-2" {
		t.Fatalf("next started = %q, want scrape-2", got)
	}
}

func TestRunnerSchedulesRepeatedWork(t *testing.T) {
	started := make(chan string, 1)
	scheduledWorked = started
	defer func() { scheduledWorked = nil }()

	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend:      backend,
		PollInterval: time.Millisecond,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&scheduledJob{})
			r.MustSchedule(lazyjobs.Every("scheduled.tick", time.Minute, &scheduledJob{Value: "tick"}, lazyjobs.RunAt(time.Now().UTC())))
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())
	runner.Start(context.Background())

	if got := waitStarted(t, started); got != "tick" {
		t.Fatalf("scheduled job = %q, want tick", got)
	}
	records, err := backend.List(context.Background(), lazyjobs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ScheduleKey != "scheduled.tick" {
		t.Fatalf("records = %#v, want scheduled.tick record", records)
	}
}

func TestRunnerRejectsSchedulesWithoutSchedulerBackend(t *testing.T) {
	_, err := lazyjobs.New(lazyjobs.Config{
		Backend: noScheduleBackend{Backend: inmemoryjobs.New()},
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&scheduledJob{})
			r.MustSchedule(lazyjobs.Every("scheduled.tick", time.Minute, &scheduledJob{Value: "tick"}))
		},
	})
	if err == nil {
		t.Fatal("New succeeded with scheduled jobs on non-scheduler backend")
	}
	if !strings.Contains(err.Error(), "SchedulerBackend") {
		t.Fatalf("error = %v, want SchedulerBackend", err)
	}
}

func TestRunnerSkipsOverlappingScheduleRuns(t *testing.T) {
	started := make(chan string, 4)
	release := make(chan struct{})
	blockingWorked = blockingJobState{started: started, release: release}
	defer func() { blockingWorked = blockingJobState{} }()

	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend:      backend,
		Workers:      1,
		PollInterval: time.Millisecond,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&blockingJob{})
			r.MustSchedule(lazyjobs.Every("scheduled.blocking", time.Millisecond, &blockingJob{Name: "scheduled"}))
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())
	defer close(release)
	runner.Start(context.Background())

	if got := waitStarted(t, started); got != "scheduled" {
		t.Fatalf("scheduled job = %q, want scheduled", got)
	}
	time.Sleep(25 * time.Millisecond)
	records, err := backend.List(context.Background(), lazyjobs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("record count = %d, want one active scheduled job: %#v", len(records), records)
	}
}

func TestRunnerDiscardsUnknownJob(t *testing.T) {
	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{Backend: backend, PollInterval: time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := backend.Insert(context.Background(), lazyjobs.InsertParams{
		Kind:        "missing",
		Queue:       lazyjobs.DefaultQueue,
		Payload:     []byte(`{}`),
		MaxAttempts: 1,
	}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())
	waitForState(t, backend, lazyjobs.StateDiscarded)
}

func TestRunnerRetriesAndDiscards(t *testing.T) {
	backend := inmemoryjobs.New()
	runner, err := lazyjobs.New(lazyjobs.Config{
		Backend:      backend,
		PollInterval: time.Millisecond,
		Define: func(r *lazyjobs.JobRunner) {
			r.MustRegister(&alwaysFailJob{})
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer runner.Stop(context.Background())

	if _, err := runner.Enqueue(context.Background(), &alwaysFailJob{}); err != nil {
		t.Fatal(err)
	}
	runner.Start(context.Background())
	waitForState(t, backend, lazyjobs.StateDiscarded)
}

type alwaysFailJob struct {
	lazyjobs.BaseJob
}

func (*alwaysFailJob) Kind() string                           { return "test.fail" }
func (*alwaysFailJob) Work(context.Context) error             { return errors.New("boom") }
func (*alwaysFailJob) JobMaxAttempts() int                    { return 2 }
func (*alwaysFailJob) JobRetryDelay(int, error) time.Duration { return time.Millisecond }

type customQueueJob struct {
	testJob
}

func (*customQueueJob) Kind() string     { return "test.custom" }
func (*customQueueJob) JobQueue() string { return "custom" }
func (j *customQueueJob) Work(ctx context.Context) error {
	if testWorked != nil {
		testWorked <- j.Value
	}
	return nil
}

type blockingJob struct {
	lazyjobs.BaseJob
	Name      string `json:"name"`
	QueueName string `json:"queue_name,omitempty"`
}

func (*blockingJob) Kind() string { return "test.blocking" }

func (j *blockingJob) JobQueue() string {
	if j.QueueName != "" {
		return j.QueueName
	}
	return lazyjobs.DefaultQueue
}

func (j *blockingJob) Work(context.Context) error {
	if blockingWorked.started != nil {
		blockingWorked.started <- j.Name
	}
	if blockingWorked.release != nil {
		<-blockingWorked.release
	}
	return nil
}

type blockingJobState struct {
	started chan string
	release chan struct{}
}

var blockingWorked blockingJobState

type scheduledJob struct {
	lazyjobs.BaseJob
	Value string `json:"value"`
}

func (*scheduledJob) Kind() string { return "test.scheduled" }

func (j *scheduledJob) Work(context.Context) error {
	if scheduledWorked != nil {
		scheduledWorked <- j.Value
	}
	return nil
}

var scheduledWorked chan string

type noScheduleBackend struct {
	lazyjobs.Backend
}

func waitStarted(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatal("job did not start")
	}
	return ""
}

func assertNoStart(t *testing.T, ch <-chan string, timeout time.Duration) {
	t.Helper()
	select {
	case got := <-ch:
		t.Fatalf("unexpected start %q", got)
	case <-time.After(timeout):
	}
}

func waitForState(t *testing.T, backend *inmemoryjobs.Backend, state lazyjobs.State) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		records, err := backend.List(context.Background(), lazyjobs.ListOptions{})
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if record.State == state {
				return
			}
		}
		select {
		case <-deadline:
			t.Fatalf("state %s not reached; records=%#v", state, records)
		case <-time.After(time.Millisecond):
		}
	}
}
