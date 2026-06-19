package lazydispatch

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"golazy.dev/lazysse"
)

func TestETagAddsValidatorToBufferedOKResponse(t *testing.T) {
	handler := ResponseBuffer().Handler(ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "hello")
	})))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", response.Body.String())
	}
	if got, want := response.Header().Get("ETag"), testETag("hello"); got != want {
		t.Fatalf("ETag = %q, want %q", got, want)
	}
}

func TestETagReturnsNotModifiedForMatchingValidator(t *testing.T) {
	etag := testETag("hello")
	handler := ResponseBuffer().Handler(ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "hello")
	})))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", etag)
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotModified)
	}
	if response.Body.String() != "" {
		t.Fatalf("body = %q, want empty", response.Body.String())
	}
	if got := response.Header().Get("ETag"); got != etag {
		t.Fatalf("ETag = %q, want %q", got, etag)
	}
	if got := response.Header().Get("Cache-Control"); got != "public, max-age=60" {
		t.Fatalf("Cache-Control = %q, want public, max-age=60", got)
	}
	if got := response.Header().Get("Content-Type"); got != "" {
		t.Fatalf("Content-Type = %q, want empty on 304", got)
	}
}

func TestETagHonorsExistingValidator(t *testing.T) {
	handler := ResponseBuffer().Handler(ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"custom"`)
		_, _ = fmt.Fprint(w, "hello")
	})))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := response.Header().Get("ETag"); got != `"custom"` {
		t.Fatalf("ETag = %q, want custom", got)
	}

	response = httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("If-None-Match", `W/"custom"`)
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotModified)
	}
}

func TestETagSkipsIneligibleResponses(t *testing.T) {
	tests := []struct {
		name   string
		method string
		write  func(http.ResponseWriter)
	}{
		{
			name:   "post",
			method: http.MethodPost,
			write: func(w http.ResponseWriter) {
				_, _ = fmt.Fprint(w, "hello")
			},
		},
		{
			name:   "created",
			method: http.MethodGet,
			write: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusCreated)
				_, _ = fmt.Fprint(w, "hello")
			},
		},
		{
			name:   "no store",
			method: http.MethodGet,
			write: func(w http.ResponseWriter) {
				w.Header().Set("Cache-Control", "private, no-store")
				_, _ = fmt.Fprint(w, "hello")
			},
		},
		{
			name:   "compressed",
			method: http.MethodGet,
			write: func(w http.ResponseWriter) {
				w.Header().Set("Content-Encoding", "gzip")
				_, _ = fmt.Fprint(w, "hello")
			},
		},
		{
			name:   "set cookie",
			method: http.MethodGet,
			write: func(w http.ResponseWriter) {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
				_, _ = fmt.Fprint(w, "hello")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := ResponseBuffer().Handler(ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				tt.write(w)
			})))

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tt.method, "/", nil))

			if got := response.Header().Get("ETag"); got != "" {
				t.Fatalf("ETag = %q, want empty", got)
			}
		})
	}
}

func TestETagCanRunWithoutSeparateResponseBuffer(t *testing.T) {
	handler := ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "hello")
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.String() != "hello" {
		t.Fatalf("body = %q, want hello", response.Body.String())
	}
	if got, want := response.Header().Get("ETag"), testETag("hello"); got != want {
		t.Fatalf("ETag = %q, want %q", got, want)
	}
}

func TestETagSkipsStreamingResponses(t *testing.T) {
	handler := ResponseBuffer().Handler(ETag().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream, err := lazysse.Start(w, r)
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		if err := stream.Send(lazysse.Event{Data: []string{"hello"}}); err != nil {
			t.Fatal(err)
		}
	})))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := response.Header().Get("ETag"); got != "" {
		t.Fatalf("ETag = %q, want empty", got)
	}
	if got, want := response.Body.String(), "data: hello\n\n"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestETagListMatching(t *testing.T) {
	if !etagMatches(`"first", W/"second"`, `"second"`) {
		t.Fatal("weak validator in list did not match")
	}
	if !etagMatches("*", `"anything"`) {
		t.Fatal("wildcard did not match")
	}
	if etagMatches(`"first", "second"`, `"third"`) {
		t.Fatal("unexpected ETag match")
	}
}

func testETag(body string) string {
	sum := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%q", fmt.Sprintf("%x", sum[:]))
}
