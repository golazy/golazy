// Package lazyhttp provides an http server compatible with lazyapp.
package lazyhttp

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"

	"golazy.dev/lazycontext"
	"golazy.dev/lazyservice"
)

type HTTPService struct {
	l *slog.Logger
	http.Server
}

type errSlog2Log struct {
	*slog.Logger
}

func (l *errSlog2Log) Write(p []byte) (n int, err error) {
	var pcs [1]uintptr
	runtime.Callers(7, pcs[:]) // skip [Callers, Infof]
	// TODO: check if 7 is always the case

	r := slog.NewRecord(time.Now(), slog.LevelError, string(p), pcs[0])

	l.Handler().Handle(context.Background(), r)
	//l.Error(string(p))
	return len(p), nil
}

func (s *HTTPService) Run(ctx context.Context) error {
	s.l = lazycontext.Get[*slog.Logger](ctx)
	if s.l == nil {
		s.l = slog.Default()
	}
	s.BaseContext = func(listener net.Listener) context.Context {
		return ctx
	}
	s.ReadHeaderTimeout = time.Millisecond * 200
	s.ErrorLog = log.New(&errSlog2Log{s.l}, "", 0)
	s.DisableGeneralOptionsHandler = true

	errCh := make(chan error)
	go func() {
		<-ctx.Done()
		s.l.Info("http server shutting down")
		sctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		errCh <- s.Shutdown(sctx)
	}()

	url := s.Addr
	if strings.HasPrefix(s.Addr, ":") {
		url = "localhost" + s.Addr
	}
	url = "http://" + url

	s.l.Info("http server starting", "addr", s.Addr, "url", url)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		return err
	}
	err := <-errCh
	if err == nil {
		s.l.Info("http server stopped")
		return nil
	}
	s.l.Error(err.Error())
	return err

}

type serviceDesc struct {
	name string
}

func (d serviceDesc) Name() string {
	return d.name
}

func (s *HTTPService) Desc() lazyservice.ServiceDescription {
	return serviceDesc{name: "lazyhttp"}
}
