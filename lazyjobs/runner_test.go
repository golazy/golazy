package lazyjobs_test

import (
	"context"
	"errors"
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
