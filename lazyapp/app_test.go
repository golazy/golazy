package lazyapp_test

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"testing"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydev"
)

type PagesController struct {
}

func (c *PagesController) Index(ctx lazyaction.Context) {
	ctx.WriteString("Hello")
}

func TestLazyApp(t *testing.T) {

	app := lazyapp.App{
		Server: &lazydev.Server{
			HTTPAddr: ":2000",
		},
	}

	running := make(chan (struct{}))
	go func() {
		err := app.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Error(err)
		}
		close(running)

	}()
	runtime.Gosched()

	res, err := http.Get("http://localhost:8080/")
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "" {
		t.Error(string(data))
	}

	app.Shutdown(context.Background())
	<-running
}
