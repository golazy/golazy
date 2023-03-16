package app

import (
	"strings"
	"testing"
	"time"
)

type eventList []Event

func (e eventList) Events() []string {
	strings := make([]string, len(e))

	for i, ev := range e {
		strings[i] = ev.Name()
	}
	return strings
}

func TestApp(t *testing.T) {

	app := New(GoAppOptions{
		Dir: "test",
	})

	defer app.Clean()

	eventList := make(eventList, 0, 100)

	combine := make([]string, 0)
	stdout := make([]string, 0)
	stderr := make([]string, 0)

	go func() {
		for {
			event, ok := <-app.Events
			if !ok {
				return
			}
			eventList = append(eventList, event)
			switch e := event.(type) {
			case EventAppStderr:
				stderr = append(stderr, string(e.Data))
				combine = append(combine, string(e.Data))
			case EventAppStdout:
				stdout = append(stdout, string(e.Data))
				combine = append(combine, string(e.Data))
			}

		}
	}()

	app.Start()

	time.Sleep(time.Second)
	app.Start()
	time.Sleep(time.Second)
	app.Stop()
	time.Sleep(time.Second / 10)

	events := strings.Split("build_start build_finish build_success app_start app_stdout app_stderr build_start build_finish build_success app_stopping app_stdout app_stop app_start app_stdout app_stderr app_stdout app_stop", " ")

	if equal(events, eventList.Events()) {
		t.Fatal("Wrong events")
	}

}

func equal(a, b []string) bool {
	return strings.Join(a, "") == strings.Join(b, "")
}
