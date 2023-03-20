package devserver

import (
	"fmt"
	"os"
	"path/filepath"

	"golazy.dev/lazydev/devserver/events"
)

type Options struct {
	BuildDir  string
	RootDir   string
	BuildArgs []string
	Events    func(events.Event)
	RunEnv    []string
	RunArgs   []string
}

type Server struct {
	opts  Options
	close chan chan error
}

func New(opts Options) *Server {
	s := &Server{
		opts:  opts,
		close: make(chan (chan (error))),
	}
	abs, err := filepath.Abs(s.opts.BuildDir)
	if err == nil {
		s.opts.BuildDir = abs
	}
	return s
}

type state struct {
	file string
	err  error
}

type action func(state) (state, action)

func (srv *Server) Close() error {
	c := make(chan error)
	srv.close <- c
	return <-c
}

func (srv *Server) Serve() error {

	s := state{}
	action := srv.build

	for action != nil {
		s, action = action(s)
	}

	return nil
}

func (s *Server) notify(e events.Event) {
	if s.opts.Events != nil {
		s.opts.Events(e)
	}
}

func (srv *Server) build(s state) (state, action) {
	if s.file != "" {
		os.Remove(s.file)
	}
	srv.notify(events.BuildStart{})
	file, out, err := build(buildOpts{
		Dir:  srv.opts.BuildDir,
		Args: srv.opts.BuildArgs,
	})
	if err != nil {
		srv.notify(events.BuildError{Out: out})
		return s, srv.standby
	}
	s.file = file
	srv.notify(events.BuildSuccess{})
	return s, srv.app_start
}

func (srv *Server) app_start(s state) (state, action) {

	outC, errC, exit, kill, err := run(runOpts{
		Path: s.file,
		Dir:  srv.opts.RootDir,
		Args: srv.opts.RunArgs,
		Env:  srv.opts.RunEnv,
	})
	if err != nil {
		srv.notify(events.AppStartError{Err: err})
		s.err = err
		return s, srv.standby
	}

	changes, close, err := watch(srv.opts.BuildDir)
	if err != nil {
		panic(err)
	}
	defer close()

	go func() {
		for stdout := range outC {
			srv.notify(events.Stdout(stdout))
		}
	}()
	go func() {
		for stderr := range errC {
			srv.notify(events.Stderr(stderr))
		}
	}()

	srv.notify(events.AppStart{})

	select {
	case c := <-srv.close:
		kill()
		<-exit
		srv.notify(events.AppStop{Reason: "close", Expected: false})
		c <- nil
		return s, nil
	case c := <-changes:
		srv.notify(events.FSChange{Changes: &c})
		kill()
		<-exit
		srv.notify(events.AppStop{Reason: "fs", Expected: true})
		return s, srv.build
	case exitStatus := <-exit:
		reason := fmt.Sprintf("Exit with code %d", exitStatus)
		srv.notify(events.AppStop{Reason: reason, Expected: false})
		return s, srv.standby
	}
}

func (srv *Server) standby(s state) (state, action) {
	srv.notify(events.Standby{Err: s.err})
	s.err = nil

	if s.file != "" {
		os.Remove(s.file)
	}

	changes, close, err := watch(srv.opts.BuildDir)
	if err != nil {
		panic(err)
	}
	defer close()
	select {
	case c := <-srv.close:
		c <- nil
		return s, nil
	case cset := <-changes:
		srv.notify(events.FSChange{Changes: &cset})
		return s, srv.build
	}
}
