package lazyservice

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/golazy/golazy/lazycontext"
)

func TestServiceFunc(t *testing.T) {

	service := func(ctx context.Context, l *slog.Logger) error {
		l.Info("hi")
		return fmt.Errorf("hi")
	}
	srv := ServiceFunc("basic", service)

	if srv.Desc().Name() != "basic" {
		t.Error(srv.Desc().Name())
	}

	err := srv.Run(lazycontext.New())
	if err.Error() != "hi" {
		t.Error("error didn't said hi")
	}
}

type testStruct string

func TestLazyService(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	app := New()

	app.AddService(ServiceFunc("http", func(ctx context.Context, l *slog.Logger) error {

		s := &http.Server{
			Addr: ":8083",
		}

		idleConnsClosed := make(chan struct{})
		go func() {
			defer close(idleConnsClosed)
			<-ctx.Done()
			l.InfoContext(ctx, "shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			err := s.Shutdown(ctx)
			if err == nil || err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			l.ErrorContext(ctx, err.Error(), "err", err)
		}()

		l.InfoContext(ctx, "listening on 8083")
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			return err
		}
		<-idleConnsClosed
		return nil

	}))

	err := app.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}

}
