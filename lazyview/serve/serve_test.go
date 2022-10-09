package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/golazy/golazy/lazyview/html"
)

func TestServe(t *testing.T) {

	http.Handle("/", Page(Html(Lang("es"), Body(H1("hola")))))

	server := httptest.NewServer(nil)
	defer server.Close()
	client := server.Client()
	res, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `<!DOCTYPE html>\n<html lang=es>\n  <body>\n    <h1>hola</h1>\n\n` {
		t.Errorf("Got %q", string(body))
	}
}
