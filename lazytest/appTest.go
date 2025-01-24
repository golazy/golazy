package lazytest

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"testing"

	"golazy.dev/lazyapp"
	"golazy.dev/lazydispatch"
	"golazy.dev/lazyservice"
)

// NewAppTest creates a new AppTest
func NewAppTest(t *testing.T, app *lazyapp.GoLazyApp) *AppTest {
	t.Helper()
	newApp := *app
	at := &AppTest{t: t, app: &newApp}
	return at
}

// AppTest is a test helper for a lazyapp
type AppTest struct {
	Ctx    context.Context
	t      *testing.T
	app    *lazyapp.GoLazyApp
	appCtx context.Context
	l      sync.Mutex
}

// Returns the application handler of lazydispatch
func (at *AppTest) Handler() http.Handler {
	at.boot()
	return at.app.LazyDispatch
}

func (at *AppTest) ctx() context.Context {
	if at.Ctx != nil {
		return at.Ctx
	}
	return context.Background()
}

// boot boots the app if there is any error it will panic. It will set appCtx
func (at *AppTest) boot() {
	at.l.Lock()
	defer at.l.Unlock()
	if at.appCtx != nil {
		return
	}

	at.app.LazyService = lazyservice.New()
	appCtxCh := make(chan context.Context)
	at.app.AddService(lazyservice.ServiceFunc("test", func(ctx context.Context, _ *slog.Logger) error {
		appCtxCh <- ctx
		if done := ctx.Done(); done != nil {
			<-done
		}
		return nil
	}))

	at.app.Start(at.ctx())

	at.appCtx = <-appCtxCh
}

func (at *AppTest) Routes() []*lazydispatch.Route {
	at.boot()
	return at.app.LazyDispatch.Routes
}

func (at *AppTest) PathFor(args ...any) string {
	at.t.Helper()
	at.boot()
	return at.app.LazyDispatch.PathFor(args...)
}
