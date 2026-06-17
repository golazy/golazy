package middlewares

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMethodOverrideChangesPostFormMethodAndReplaysBody(t *testing.T) {
	var gotMethod string
	var gotOriginal string
	var gotBody string
	handler := MethodOverride().Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotOriginal = OriginalMethod(r)
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
	}))

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_method=patch&name=Ada"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotMethod != http.MethodPatch {
		t.Fatalf("method = %q, want PATCH", gotMethod)
	}
	if gotOriginal != http.MethodPost {
		t.Fatalf("original method = %q, want POST", gotOriginal)
	}
	if gotBody != "_method=patch&name=Ada" {
		t.Fatalf("body = %q, want replayed body", gotBody)
	}
}

func TestMethodOverrideRejectsInvalidMethod(t *testing.T) {
	handler := MethodOverride().Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called")
	}))
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_method=trace"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMethodOverrideIgnoresNonPostAndUpgradeRequests(t *testing.T) {
	for _, test := range []struct {
		name    string
		request *http.Request
	}{
		{
			name:    "get",
			request: httptest.NewRequest(http.MethodGet, "/", strings.NewReader("_method=delete")),
		},
		{
			name: "upgrade",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_method=delete"))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Upgrade", "websocket")
				return req
			}(),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := MethodOverride().Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != test.request.Method {
					t.Fatalf("method = %q, want %q", r.Method, test.request.Method)
				}
			}))
			handler.ServeHTTP(httptest.NewRecorder(), test.request)
			if !called {
				t.Fatal("next handler was not called")
			}
		})
	}
}

func TestMethodOverrideReadsMultipartMethodAndReplaysBody(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	method, err := writer.CreateFormField("_method")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := method.Write([]byte("delete")); err != nil {
		t.Fatal(err)
	}
	name, err := writer.CreateFormField("name")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := name.Write([]byte("Ada")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	var gotMethod string
	var gotBody []byte
	handler := MethodOverride().Handler(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotBody, _ = io.ReadAll(r.Body)
	}))
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %q, want DELETE", gotMethod)
	}
	if !bytes.Equal(gotBody, body.Bytes()) {
		t.Fatal("body was not replayed exactly")
	}
}
