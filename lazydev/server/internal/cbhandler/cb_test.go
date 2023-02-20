package cbhandler

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCB(t *testing.T) {

	url, close, err := startHelloServer()
	if err != nil {
		t.Fatal(err)
	}
	defer close()

	fallbackHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("fallback"))
	})

	cb := New(fallbackHandler)

	server := httptest.NewServer(cb)
	defer server.Close()

	expect(t, server, "fallback")
	cb.Open(url)
	expect(t, server, "hello")
	cb.Close()
	expect(t, server, "fallback")

}

func expect(t *testing.T, server *httptest.Server, expectation string) {
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != expectation {
		t.Fatalf("expected %s, got %s", expectation, string(body))
	}

}

func startHelloServer() (addr *url.URL, close func(), err error) {

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return nil, nil, err
	}

	port := l.Addr().(*net.TCPAddr).Port
	url, err := url.Parse(fmt.Sprintf("http://localhost:%d/", port))
	if err != nil {
		return nil, nil, err
	}

	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		}),
	}

	go s.Serve(l)

	return url, func() {
		s.Shutdown(context.Background())
	}, nil
}
