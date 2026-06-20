package lazycontroller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeParsesSubmittedForm(t *testing.T) {
	type postForm struct {
		Title   string `schema:"title"`
		Count   int    `schema:"count"`
		Ignored string `schema:"ignored"`
	}

	request := httptest.NewRequest(http.MethodPost, "/posts?title=query", strings.NewReader("title=Hello&count=3&extra=ignored"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	base := Base{request: request}

	var form postForm
	if err := base.Decode(&form); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if form.Title != "Hello" {
		t.Fatalf("Title = %q, want %q", form.Title, "Hello")
	}
	if form.Count != 3 {
		t.Fatalf("Count = %d, want 3", form.Count)
	}
}

func TestDecodeZeroEmpty(t *testing.T) {
	type postForm struct {
		Count int `schema:"count"`
	}

	request := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader("count="))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	base := Base{request: request}
	form := postForm{Count: 42}

	if err := base.Decode(&form); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if form.Count != 0 {
		t.Fatalf("Count = %d, want 0", form.Count)
	}
}

func TestDecodeWrapsDecodeErrorsAsBadRequest(t *testing.T) {
	type postForm struct {
		Count int `schema:"count"`
	}

	request := httptest.NewRequest(http.MethodPost, "/posts", strings.NewReader("count=abc"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	base := Base{request: request}

	var form postForm
	err := base.Decode(&form)
	if err == nil {
		t.Fatal("Decode returned nil error")
	}

	var httpError *HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("Decode error type = %T, want *HTTPError", err)
	}
	if httpError.Status != http.StatusBadRequest {
		t.Fatalf("HTTP status = %d, want %d", httpError.Status, http.StatusBadRequest)
	}
}
