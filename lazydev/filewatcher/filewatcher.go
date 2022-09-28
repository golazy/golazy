// Package filewatcher notifies when the filesystem has change.
// It goes up to the top directory that holds a go.mod file
package filewatcher

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/dietsche/rfsnotify"
	"gopkg.in/fsnotify.v1"
)

// Op holds the operation name
type Op fsnotify.Op

// String return the operation name Create , Write , Remove , Rename or Chmod
func (o Op) String() string {
	return fsnotify.Op(o).String()
}

// Change represent a change in the filesystem
type Change struct {
	Path string // Path is the path that changed
	Op   Op     // Op is the change operation
}

// ChangeSet is a collection of changes
type ChangeSet []Change

// FileWatcher looks for changes in the top most directory that have a go.mod
type FileWatcher struct {
	topDir string
	c      chan (ChangeSet)
	w      *rfsnotify.RWatcher
}

// New initializes a FileWatcher in the given directory
// It will go up to the top most directory that holds a go.mod
func New(dir string) (*FileWatcher, error) {
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("filepath is not absolute")
	}
	topDir, err := findRootDirectory(dir)
	if err != nil {
		return nil, err
	}
	fw := &FileWatcher{
		topDir: topDir,
	}

	return fw, nil
}

// Close stop listening for changes in the file system
// Once close, the channel will be closed
func (fw *FileWatcher) Close() error {
	return fw.w.Close()
}

// Watch start watching for recursively in the project
func (fw *FileWatcher) Watch() (<-chan (ChangeSet), error) {
	if fw.c != nil {
		return nil, fmt.Errorf("Watch was called more than once")
	}

	fw.c = make(chan (ChangeSet), 100)

	watcher, err := rfsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	fw.w = watcher
	fw.w.AddRecursive(fw.topDir)

	go fw.wait()

	return fw.c, nil
}

// IgnoredFiles is a list of files that should not trigger a change
var IgnoredFiles = []string{}

// IgnoredDirs is a list of directories that should not tirgger a change
var IgnoredDirs = []string{".git"}

func (fw *FileWatcher) shouldIgnore(e fsnotify.Event) bool {
	changedPath := e.Name
	for _, file := range IgnoredFiles {
		if path.Base(changedPath) == file {
			return true
		}
	}

	dir := changedPath

	// Be nice and not hit the disk constantly
	// Side effect: A change in a file that is called the same as an ignoredDir is ignored
	//
	//fileInfo, err := os.Stat(dir)
	//if err != nil || !fileInfo.IsDir() {
	//	dir = path.Dir(dir)
	//}

	for _, ignoredDir := range IgnoredDirs {
		for ; len(dir) > len(fw.topDir); dir = path.Dir(dir) {
			if path.Base(dir) == ignoredDir {
				return true
			}
		}

	}

	return false
}

var delay time.Duration = 100

func (fw *FileWatcher) wait() {
	wEvents := fw.w.Events
	wErrors := fw.w.Errors
	eventBuffer := make(ChangeSet, 0)
	var timeOut <-chan (time.Time)

	for wEvents != nil || wErrors != nil {
		select {
		case event, ok := <-wEvents:
			if !ok {
				wEvents = nil
				continue
			}
			if fw.shouldIgnore(event) {
				continue
			}
			eventBuffer = append(eventBuffer, Change{Path: event.Name, Op: Op(event.Op)})
			timeOut = time.After(time.Millisecond * delay)
		case error, ok := <-wErrors:
			if !ok {
				wErrors = nil
				continue
			}
			log.Println(error)
		case _, ok := <-timeOut:
			fw.c <- eventBuffer
			eventBuffer = make(ChangeSet, 0)
			if !ok {
				timeOut = nil
			}
		}
	}
	if len(eventBuffer) > 0 {
		fw.c <- eventBuffer
	}
	close(fw.c)
}

func findRootDirectory(start string) (string, error) {
	topDir := ""

	for dir := start; path.Dir(dir) != dir; dir = path.Dir(dir) {
		gomod := path.Join(dir, "go.mod")
		_, err := os.Stat(gomod)
		if err == nil {
			topDir = dir
			continue
		}
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		return "", err
	}

	return topDir, nil
}
