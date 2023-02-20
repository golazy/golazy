package server_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazydev/server"
)

func TestServer(t *testing.T) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	s := server.Server{
		BuildDir:      "test_server",
		BuildArgs:     strings.Split("-buildvcs=false", " "), // Required while https://go-review.googlesource.com/c/go/+/463849 is solved
		HttpHandler:   StringHandler("http"),
		PrefixHandler: StringHandler("golazy"),
		FallbackHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Refresh", "1")
			w.Write([]byte("fallback"))
		}),
		Addr: "127.0.0.1:34412",
	}

	go s.ListenAndServe()

	expect(t, "http://localhost:34412", "http")
	expect(t, "https://localhost:34412", "backend")
	expect(t, "https://localhost:34412/golazy", "golazy")

	os.Create("test_server/main.go")
	defer os.Remove("test_server/main.go")

	time.Sleep(1 * time.Second)
	expect(t, "https://localhost:34412", "fallback")

	os.Remove("test_server/main.go")
	time.Sleep(1 * time.Second)
	expect(t, "https://localhost:34412", "backend")

}

type StringHandler string

func (h StringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(h))
}

func expect(t *testing.T, url, expectation string) {
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
