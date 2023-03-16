package app

import (
	"fmt"
	"time"
)

type Event interface {
	Event() string
	Name() string
	CreatedAt() time.Time
}

func newEvent(name string) EventBase {
	return EventBase{
		Type: name,
		Time: time.Now(),
	}
}

type EventBase struct {
	Type string
	Time time.Time
}

func (e EventBase) Name() string {
	return e.Type
}

func (e EventBase) CreatedAt() time.Time {
	return e.Time
}

func (e EventBase) Event() string {
	return fmt.Sprintf("%v: %s", e.Time, e.Type)
}

type EventAppStart struct {
	EventBase
	Pid     int
	Environ []string
}

type EventAppStartFail struct {
	EventBase
	Err error
}

type EventAppStdout struct {
	EventBase
	Data []byte
}

type EventAppStderr struct {
	EventBase
	Data []byte
}

type EventAppStopping struct {
	EventBase
	Pid int
}

type EventAppStop struct {
	EventBase
	ExitCode int
	Pid      int
	Success  bool
}

// Uninplemented
type EventAppUnexpectedExit struct {
	EventBase
}

type EventAppBuildStart struct {
	EventBase
}

// EventAppBuildFinish fires when the build finished regarding the build status
type EventAppBuildFinish struct {
	EventBase
	Err error
	Out []byte
}

type EventAppBuildFailure struct {
	EventBase
	Out []byte
	Err error
}

type EventAppBuildSuccess struct {
	EventBase
}
