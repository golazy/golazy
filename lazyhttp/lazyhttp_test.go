package lazyhttp

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

type mykey string

func TestServer(t *testing.T) {
	addr := "localhost:8085"
	s := &HTTPService{}
	s.Addr = addr
	s.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(mykey("key")).(string)
		w.Write([]byte(v))
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, mykey("key"), "Hello, world!")

	errCh := make(chan error)

	go func() {
		errCh <- s.Run(ctx)
	}()

	resp, err := http.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "Hello, world!" {
		t.Fatalf("unexpected response: %s", body)
	}

	if err = <-errCh; err != nil {
		t.Fatal(err)
	}

}
