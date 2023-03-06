package portalserver

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"golazy.dev/lazydev/devserver"
	"golazy.dev/lazydev/devserver/events"
)

type DevApp interface {
	http.Handler
	Event(events.Event)
	ListenAndServe(addr string) error
	Close() error
}

type Options struct {
	App       DevApp
	Addr      string
	BuildArgs []string
	BuildDir  string
}

type Server struct {
	opts    Options
	dev     *devserver.Server
	devAddr string
	close   chan chan error
}

func New(opts Options) *Server {
	if opts.App == nil {
		panic("no app set")
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:2000"
	}
	if opts.BuildDir == "" {
		panic("no build dir set")
	}

	devAddr := getFreeAddr()

	s := &Server{
		opts:    opts,
		close:   make(chan (chan error)),
		devAddr: devAddr,
	}

	s.dev = devserver.New(devserver.Options{
		BuildDir:  opts.BuildDir,
		BuildArgs: opts.BuildArgs,
		RunEnv:    getRunEnv(devAddr),
		Events:    s.event,
	})
	return s
}

func (s *Server) ListenAndServe() error {
	devEnd := make(chan error)
	httpEnd := make(chan error)

	go func() { devEnd <- s.dev.Serve() }()
	go func() { httpEnd <- s.opts.App.ListenAndServe(s.opts.Addr) }()

	select {
	case err := <-devEnd:
		s.opts.App.Close()
		<-httpEnd
		return err
	case err := <-httpEnd:
		return err
	case c := <-s.close:
		err1 := s.opts.App.Close()
		err2 := s.dev.Close()
		<-devEnd
		<-httpEnd
		err := errors.Join(net.ErrClosed, err1, err2)
		c <- err
		return err
	}

}

func (s *Server) Close() error {
	c := make(chan error)
	s.close <- c
	return <-c
}

func (s *Server) event(e events.Event) {
	if appStart, ok := e.(events.AppStart); ok {
		u, err := url.Parse("http://" + s.devAddr)
		if err != nil {
			panic(err)
		}
		appStart.URL = u
		e = appStart
	}

	s.opts.App.Event(e)
}

func getFreeAddr() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func getRunEnv(addr string) []string {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		panic("invalid addr")
	}
	return []string{"LISTEN=" + addr, "PORT=" + parts[1]}
}
