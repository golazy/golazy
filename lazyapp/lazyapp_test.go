package lazyapp

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/golazy/golazy/lazyassets"
	"github.com/golazy/golazy/lazycontext"
	"github.com/golazy/golazy/lazyservice"
	"github.com/golazy/golazy/lazyview"
)

func TestAppBuilder(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app := New("test", "1.0.0")
	app.LazyAssets.AddFile("index.html", []byte("Hello, World!"))

	errCh := app.Start(ctx)

	resp, err := http.Get("http://localhost:2000/index.html")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Hello, World!" {
		t.Errorf("Expected: Hello, World! Got: %s", string(body))
	}
	cancel()

	if err := <-errCh; err != nil {
		t.Errorf("Error: %v", err)
	}
}

func TestAppHasContexts(t *testing.T) {
	app := New("test", "1.0.0")

	app.AddService(lazyservice.ServiceFunc("config", func(ctx context.Context, l *slog.Logger) error {
		if value := lazycontext.Get[lazycontext.LazyContext](ctx); value == nil {
			t.Fatal("*lazycontext.AppContext is nil")
		}
		if value := lazycontext.Get[*GoLazyApp](ctx); value == nil {
			t.Fatal("*GoLazyApp is nil")
		}
		if value := lazycontext.Get[lazyservice.Manager](ctx); value == nil {
			t.Fatal("lazyservice.Manager is nil")
		}

		if value := lazycontext.Get[*lazyassets.Server](ctx); value == nil {
			t.Fatal("lazyassets.Server is nil")
		}
		if value := lazycontext.Get[*lazyassets.Storage](ctx); value == nil {
			t.Fatal("storage is nil")
		}

		if value := lazycontext.Get[*lazyview.Views](ctx); value == nil {
			t.Fatal("lazyview.Views is nil")
		}

		return nil
	}))

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	app.Run(ctx)

}
