package devserver

import (
	"fmt"
	"os"
	"testing"

	"golazy.dev/lazydev/devserver/events"
)

func TestServer(t *testing.T) {
	os.Remove("test_server/main.go")
	defer os.Remove("test_server/main.go")

	eCtoFilter := make(chan events.Event, 10000)
	eCAll := make(chan events.Event)
	es := make([]events.Event, 0)

	go func() {
		for e := range eCAll {
			es = append(es, e)
			eCtoFilter <- e
		}
	}()

	s := New(Options{
		BuildDir:  "test_server",
		BuildArgs: []string{"-buildvcs=false"},
		Events:    func(e events.Event) { eCAll <- e },
	})

	printEvents := func() {

		for _, e := range es {
			fmt.Print(e.Type(), " ")
		}
		fmt.Println("")
	}

	waitFor := func(t string) events.Event {
		for e := range eCtoFilter {
			printEvents()
			if e.Type() == t {
				fmt.Println("Found", t)
				return e
			}
		}
		return nil
	}

	go func() {
		err := s.Serve()
		if err != nil {
			t.Error(err)
		}
	}()

	waitFor("build_start")
	waitFor("build_success")
	waitFor("app_start")
	e := waitFor("stdout").(events.Stdout)
	if string(e) != "Listening on http://127.0.0.1:2001\n" {
		t.Error("Expected 'Hello World' got", string(e))
	}

	f, _ := os.Create("test_server/main.go")
	f.Close()

	waitFor("fs_change")
	waitFor("app_stop")
	waitFor("build_start")
	waitFor("build_error")
	waitFor("standby")

}
