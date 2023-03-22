package runner

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReadEvent(t *testing.T) {

	r := New(exec.Command("echo", "Listening 127.0.0.1:80"), nil)
	defer r.Close()

	r.Start()

	if _, ok := (<-r.Events).(EventStart); !ok {
		t.Fatal("Expected a EventSTart")
	}
	eventReady, ok := (<-r.Events).(EventReady)
	if !ok {
		t.Fatal("Expected a EventSTart")
	}

	if string(eventReady.Data) != "Listening 127.0.0.1:80\n" {
		t.Fatalf("%q", string(eventReady.Data))
	}

	if _, ok := (<-r.Events).(EventStopped); !ok {
		t.Fatal("Expected a EventSTart")
	}

}

func TestLifecycle(t *testing.T) {

	r := New(exec.Command("echo", "hello"), nil)
	defer r.Close()

	// Check defaults
	err := r.Stop()
	if err != ErrNotRunning {
		t.Fatal("expected some error")
	}

	// Start the programm
	go func() {
		err := r.Start()
		if err != nil {
			panic(err)
		}
	}()

	// Did we receive start?
	if event, ok := (<-r.Events).(EventStart); !ok {
		t.Fatal("Expected EventStart. Got:", event)
	}

	// Did we receive stop?
	event, ok := (<-r.Events).(EventStopped)
	if !ok {
		t.Fatal("expected EventStopped, got:", event)
	}
	if event.ExitCode != 0 {
		t.Error(event.ExitCode)
	}
	if string(strings.Join(event.Output, "\n")) != "hello" {
		t.Error(event.Output)
	}

	// Now lets tests restart
	r.cmd = exec.Command("sleep", "10")

	// As it is stopped, it behaves like start
	err = r.Restart()
	if err != nil {
		t.Fatal(err)
	}
	ev := <-r.Events
	if _, ok := (ev).(EventRestart); !ok {
		t.Fatalf("expected EventRestart, got: %T", ev)
	}

	err = r.Restart()
	if err != nil {
		t.Fatal(err)
	}

	ev = <-r.Events
	if _, ok := (ev).(EventRestart); !ok {
		t.Fatalf("expected EventRestart, got: %T", ev)
	}

	ev = <-r.Events
	evStopped, ok := (ev).(EventStopped)

	if !ok {
		t.Fatalf("expected EventStop, got: %T", ev)
	}

	if evStopped.ExitCode != -1 {
		t.Fatal(ev)
	}

	err = r.Stop()
	if err != nil {
		t.Fatal(err)
	}

	ev = <-r.Events
	_, ok = (ev).(EventStop)

	if !ok {
		t.Fatalf("expected EventStop, got: %T", ev)
	}

	ev = <-r.Events
	evStopped, ok = (ev).(EventStopped)

	if !ok {
		t.Fatalf("expected EventStop, got: %T", ev)
	}

	if evStopped.ExitCode != -1 {
		t.Fatal(ev)
	}

}
