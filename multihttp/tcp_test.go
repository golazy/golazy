package multihttp

import (
	"context"
	"net/http"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

type TestLogger struct {
	t     *testing.T
	group string
}

func (t *TestLogger) Enabled(context.Context, slog.Level) bool { return true }
func (t *TestLogger) Handle(ctx context.Context, r slog.Record) error {

	t.t.Logf("LOG> %15s %6s %q", t.group, r.Level, r.Message)
	return nil
}
func (t *TestLogger) WithAttrs(attrs []slog.Attr) slog.Handler { return t }
func (t *TestLogger) WithGroup(name string) slog.Handler {
	l2 := *t
	l2.group = name
	return &l2
}

func NewTestLogger(t *testing.T) {
	tl := &TestLogger{t: t}
	slog.SetDefault(slog.New(tl))
}

func TestTCP(t *testing.T) {

	NewTestLogger(t)

	s := &tcp{
		Addr: "127.0.0.1:1999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hi"))
		}),
		TLSConfig: getTLSConfig(),
	}

	done := make(chan (struct{}))
	var err error

	go func() {
		err = s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Error(err)
		}
		done <- struct{}{}
	}()

	time.Sleep(time.Second * 1)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Close()
	if err != nil {
		t.Error(err)
	}

	<-done

}
