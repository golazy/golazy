package events

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
type Listen struct {
	Addr string
}

func (e Listen) Type() string {
	return "listen"
}

func (e Listen) String() string {
	return fmt.Sprintf("listening for connections in %s", e.Addr)
}

// build_start
type BuildStart struct {
}

func (e BuildStart) String() string {
	return "building..."
}

func (e BuildStart) Type() string {
	return "build_start"
}

// build_success
type BuildSuccess struct{}

func (e BuildSuccess) String() string {
	return "build success"
}

func (e BuildSuccess) Type() string {
	return "build_success"
}

// build_error
type BuildError struct {
	Out []byte
}

func (e BuildError) String() string {
	return "build failure"
}

func (e BuildError) Type() string {
	return "build_error"
}

// build_app_start
type AppStart struct {
	URL *url.URL
}

func (e AppStart) String() string {
	if e.URL == nil {
		return "app started"
	}
	return "app started in " + e.URL.String()
}

func (e AppStart) Type() string {
	return "app_start"
}

// app_start_error
type AppStartError struct {
	Err error
}

func (e AppStartError) String() string {
	return "app start error: " + e.Err.Error()
}

func (e AppStartError) Type() string {
	return "app_start_error"
}

// app_stop
type AppStop struct {
	Expected bool
	Reason   string
}

func (e AppStop) String() string {
	return "app stopped: " + e.Reason
}

func (e AppStop) Type() string {
	return "app_stop"
}

type FSChange struct {
	Changes *filewatcher.ChangeSet
}

func (e FSChange) String() string {
	return "file system changes"
}

func (e FSChange) Type() string {
	return "fs_change"
}

type Stdout []byte

func (e Stdout) String() string {
	return string(e)
}

func (e Stdout) Type() string {
	return "stdout"
}

type Stderr []byte

func (e Stderr) String() string {
	return string(e)
}

func (e Stderr) Type() string {
	return "fs_change"
}

type Standby struct {
	Err error
}

func (e Standby) String() string {
	if e.Err == nil {
		return "standby"
	}
	return "standby: " + e.Err.Error()
}

func (e Standby) Type() string {
	return "standby"
}
