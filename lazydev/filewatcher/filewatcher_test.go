package filewatcher

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"
)

func TestFindRootDirectory(t *testing.T) {
	wd, _ := os.Getwd()
	top, err := findRootDirectory(wd)
	if err != nil {
		t.Fatal(err)
	}
	if path.Base(top) != "golazy" {
		t.Fatal(top)
	}
}

func TestWatcher(t *testing.T) {

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fw, err := New(wd)
	if err != nil {
		t.Fatal(err)
	}

	w, err := fw.Watch()
	if err != nil {
		t.Fatal(err)
	}

	os.Create("t")
	time.Sleep(time.Millisecond)
	os.Remove("t")
	time.Sleep(time.Millisecond)
	fw.Close()

	changes := make([]string, 0)
	calls := 0
	for c := range w {
		calls = calls + 1
		for _, change := range c {
			changes = append(changes,
				fmt.Sprintf("%s(%s)", change.Op, path.Base(change.Path)))

		}
	}
	changeList := strings.Join(changes, "\n")

	if calls != 1 {
		t.Fatal("expected 1 call. Got ", calls)
	}
	if changeList != "CREATE(t)\nREMOVE(t)" {
		t.Fatal(changeList)
	}

	// Test delay is working

	delay = 0
	calls = 0
	fw, err = New(wd)
	if err != nil {
		t.Fatal(err)
	}

	w, err = fw.Watch()
	if err != nil {
		t.Fatal(err)
	}

	os.Create("t")
	time.Sleep(time.Millisecond)
	os.Remove("t")
	time.Sleep(time.Millisecond)
	fw.Close()

	for range w {
		calls = calls + 1
	}
	if calls != 2 {
		t.Fatal("expecting 2 calls. Got", calls)
	}

	// Test ignore file
	IgnoredFiles = []string{"t"}
	fw, err = New(wd)
	if err != nil {
		t.Fatal(err)
	}

	w, err = fw.Watch()
	if err != nil {
		t.Fatal(err)
	}

	os.Create("t")
	time.Sleep(time.Millisecond)
	os.Remove("t")
	time.Sleep(time.Millisecond)
	fw.Close()

	calls = 0
	for range w {
		calls = calls + 1
	}
	if calls != 0 {
		t.Fatal("t should have been ignored")
	}

	// Test ignore dir
	IgnoredFiles = []string{}
	IgnoredDirs = []string{"filewatcher"}
	fw, err = New(wd)
	if err != nil {
		t.Fatal(err)
	}

	w, err = fw.Watch()
	if err != nil {
		t.Fatal(err)
	}

	os.Create("t")
	time.Sleep(time.Millisecond)
	os.Remove("t")
	time.Sleep(time.Millisecond)
	fw.Close()

	calls = 0
	for range w {
		calls = calls + 1
	}
	if calls != 0 {
		t.Fatal("filewatcher should have been ignored")
	}

}
