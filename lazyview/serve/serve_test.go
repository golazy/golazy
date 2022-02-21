package serve

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/guillermo/golazy/lazyview/html"
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
	if string(body) != `<html lang="es"><body><h1>hola</h1></body></html>` {
		t.Error("Got", string(body))
	}
}
