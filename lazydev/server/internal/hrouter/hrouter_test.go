package hrouter

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"
)

func backend() (url string, close func()) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("backend"))
	}))

	return s.URL, s.Close
}

func TestServer(t *testing.T) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	s := Server{
		HTTPHandler:     StringHandler("http"),
		PrefixHandler:   StringHandler("prefix"),
		Prefix:          "/prefix",
		FallbackHandler: StringHandler("fallback"),
		Addr:            ":54311",
	}

	backUrl, bclose := backend()
	defer bclose()

	done := make(chan (error))
	go func() {
		done <- s.ListenAndServe()
	}()
	runtime.Gosched()

	expect(t, "http://localhost:54311", "http")
	expect(t, "https://localhost:54311/", "fallback")
	expect(t, "https://localhost:54311/prefix", "prefix")

	u, _ := url.Parse(backUrl)
	s.CBOpen(u)
	expect(t, "https://localhost:54311/", "backend")
	s.CBClose()
	expect(t, "https://localhost:54311/", "fallback")
	s.Close()

	defer func() {
		err := <-done
		if err != nil {
			t.Fatal(err)
		}
	}()

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
