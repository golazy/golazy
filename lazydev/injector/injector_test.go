package injector

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseInjector(t *testing.T) {

	expect := func(expectation string, in ...string) {
		rec := httptest.NewRecorder()
		ri := &ResponseInjector{
			ResponseWriter: rec,
			content:        bytes.NewBufferString("hi"),
		}
		t.Helper()

		for _, s := range in {
			ri.Write([]byte(s))
		}
		if rec.Body.String() != expectation {
			t.Fatalf("Got %q, want %q For: %+v", rec.Body.String(), expectation, in)
		}
	}

	expectDouble := func(expectation, in string) {
		t.Helper()
		expect(expectation, in)
		expect(expectation, strings.Split(in, "")...)
	}

	// No insertion
	expectDouble("hola", "hola")

	expectDouble("hi<body", "<body")

	expectDouble("<html>hi<body></html>", "<html><body></html>")

	expectDouble("<html>hi<body id=asdf></html>", "<html><BoDy id=asdf></html>")

}

func TestInjector(t *testing.T) {

	expect := func(expectation, response string) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, r := range []byte(response) {
				w.Write([]byte{r})
			}
		})

		injector := New(bytes.NewBufferString("HI"))

		server := httptest.NewServer(injector(handler))

		res, err := server.Client().Get(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		content, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != expectation {
			t.Fatalf("Got %q, want %q", string(content), expectation)
		}

	}

	expect("hi", "hi")
	expect("HI<body", "<body")
	expect("<html> HI<body ></html>", "<html> <Body ></html>")

}
