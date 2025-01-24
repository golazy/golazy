package responsepipe

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))

	})

	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("data: hello\n\n"))
		w.Write([]byte("data: hello\n\n"))
	})
	mux.HandleFunc("/flush", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
		w.(http.Flusher).Flush()
		w.Write([]byte("hello"))
	})
	mux.HandleFunc("/hijack", func(w http.ResponseWriter, r *http.Request) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
		conn.Write([]byte("Connection: Closed\r\n"))
		conn.Write([]byte("\r\n"))
		conn.Write([]byte("hello"))
		conn.Close()
	})
	return mux
}

type EditHookHandler struct {
	Next http.Handler
	Hook func(response Response, err error)
}

func (h *EditHookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wc := New(w)
	defer wc.Close()

	h.Next.ServeHTTP(wc, r)

	response, ok := wc.(Response)
	if !ok {
		h.Hook(nil, ErrResponseIsNotEditable)
		return
	}

	if err := response.Error(); err != nil {
		h.Hook(nil, err)
		return
	}

	h.Hook(response, nil)

}

func TestResponseWriter(t *testing.T) {
	app := testHandler()

	var editHook func(r Response, err error)

	editHookHandler := &EditHookHandler{
		Next: app,
		Hook: func(r Response, err error) { editHook(r, err) },
	}

	server := httptest.NewServer(editHookHandler)

	t.Run("hello", func(t *testing.T) {
		editHook = func(r Response, err error) {
			if r.Error() != nil {
				t.Error("can't edit")
			}
			data := r.Body()
			*data = append(*data, []byte(" potato")...)
		}

		resp, err := http.Get(server.URL + "/hello")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if string(data) != "hello potato" {
			t.Fatalf("expected hello potato, got %s", data)
		}
	})

	t.Run("sse", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/sse")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if string(data) != "data: hello\n\ndata: hello\n\n" {
			t.Fatalf("expected hello, got %q", data)
		}
	})

	t.Run("flush", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/flush")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if string(data) != "hellohello" {
			t.Fatalf("expected hellohello, got %q", data)
		}
	})

	t.Run("hijack", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/hijack")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if string(data) != "hello" {
			t.Fatalf("expected hello, got %q", data)
		}
	})

}
