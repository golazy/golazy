package portalserver_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazydev/portalserver"
)

func TestServer(t *testing.T) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	app := &TestDevApp{
		E: make(chan events.Event, 10000),
	}
	s := portalserver.New(portalserver.Options{
		App:       app,
		BuildDir:  "test_server",
		BuildArgs: strings.Split("-buildvcs=false", " "), // Required while https://go-review.googlesource.com/c/go/+/463849 is solved
		Addr:      "127.0.0.1:34412",
	})
	defer func() {
		types := make([]string, 0)
		for _, e := range app.events {
			types = append(types, e.Type())
		}

		t.Log(types)
	}()

	// Ensure we have clean state
	os.Remove("test_server/main.go")

	go s.ListenAndServe()

	e := app.waitFor("app_start").(events.AppStart)
	if e.URL == nil {
		t.Fatal("no url")
	}
	expect(t, "http://localhost:34412", "proxy")

	// Fail a build
	os.Create("test_server/main.go")
	defer os.Remove("test_server/main.go")

	app.waitFor("app_stop")

	expect(t, "http://localhost:34412", "portal")

	os.Remove("test_server/main.go")

	app.waitFor("build_success")
	app.waitFor("app_start")

	expect(t, "http://localhost:34412", "proxy")

}

func expect(t *testing.T, url, expectation string) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != expectation {
		t.Fatalf("expected %s, got %s", expectation, string(body))
	}
}
