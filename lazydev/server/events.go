package server

import (
	"fmt"
	"net/url"

	"golazy.dev/lazydev/filewatcher"
)

type Event interface {
	String() string
	Type() string
}

// listen
type EventListen struct {
	Addr string
}

func (e EventListen) Type() string {
	return "listen"
}

func (e EventListen) String() string {
	return fmt.Sprintf("listening for connections in %s", e.Addr)
}

// build_start
type EventBuildStart struct {
}

func (e EventBuildStart) String() string {
	return "building..."
}

func (e EventBuildStart) Type() string {
	return "build_start"
}

// build_success
type EventBuildSuccess struct{}

func (e EventBuildSuccess) String() string {
	return "build success"
}

func (e EventBuildSuccess) Type() string {
	return "build_success"
}

// build_error
type EventBuildError struct {
	Out []byte
}

func (e EventBuildError) String() string {
	return "build failure"
}

func (e EventBuildError) Type() string {
	return "build_error"
}

// build_app_start
type EventAppStart struct {
	URL *url.URL
}

func (e EventAppStart) String() string {
	return "app started"
}

func (e EventAppStart) Type() string {
	return "app_start"
}

// app_start_error
type EventAppStartError struct {
	Err error
}

func (e EventAppStartError) String() string {
	return "app start error: " + e.Err.Error()
}

func (e EventAppStartError) Type() string {
	return "app_start_error"
}

// app_stop
type EventAppStop struct {
	Expected bool
	Reason   string
}

func (e EventAppStop) String() string {
	return "app stopped: " + e.Reason
}

func (e EventAppStop) Type() string {
	return "app_stop"
}

type EventFSChange struct {
	Changes *filewatcher.ChangeSet
}

func (e EventFSChange) String() string {
	return "file system changes"
}

func (e EventFSChange) Type() string {
	return "fs_change"
}
