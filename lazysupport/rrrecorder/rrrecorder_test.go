package rrrecorder

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(w, r.Body)
	}))
	defer server.Close()

	req, err := http.NewRequest("GET", server.URL+"/404", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}

	reqRec, err := RecordRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if reqRec.URL != req.URL.String() {
		t.Error("url not match")
	}

	if string(reqRec.RequestBody) != "hello" {
		t.Error("body not match")
	}
	if reqRec.Method != "GET" {
		t.Error("method not match")
	}
	if len(reqRec.RequestHeader) != len(req.Header) {
		t.Error("header not match")
	}
	for k, v := range reqRec.RequestHeader {
		if !reflect.DeepEqual(v, req.Header[k]) {
			t.Error("header not match", k)
		}
	}

	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	resRec, err := RecordResponse(res)
	if err != nil {
		t.Fatal(err)
	}

	if string(resRec.ResponseBody) != "hello" {
		t.Errorf("body not match. expected %q got %q", "hello\n", string(resRec.ResponseBody))

	}
	if resRec.Status != 200 {
		t.Error("status not match")
	}

	for k, v := range resRec.ResponseHeader {
		if !reflect.DeepEqual(v, res.Header[k]) {
			t.Error("header not match", k)
		}
	}

}

func TestResponseRecorder(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Name", "golazy")
		w.WriteHeader(200)
		io.Copy(w, r.Body)
	}

	req, err := http.NewRequest("GET", "/404", strings.NewReader("hello"))
	if err != nil {
		t.Fatal(err)
	}

	w := NewRecorder()

	handler(w, req)

	resRec := w.Response
	if string(resRec.ResponseBody) != "hello" {
		t.Errorf("body not match. expected %q got %q", "hello\n", string(resRec.ResponseBody))

	}
	if resRec.Status != 200 {
		t.Error("status not match")
	}

	if resRec.ResponseHeader.Get("X-Name") != "golazy" {
		t.Error("header not match")
	}

}

func TestRecorder(t *testing.T) {
	backend := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Name", "golazy")
		w.WriteHeader(200)
		io.Copy(w, r.Body)
	}

	rts := make(chan *RoundTrip, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt, _ := ServeTo(w, r, backend)
		rt.ResponseBody = []byte(strings.ToUpper(string(rt.ResponseBody)))
		rt.WriteTo(w)
		rts <- rt
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/404", strings.NewReader("hello"))
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "HELLO" {
		t.Errorf("body not match. expected %q got %q", "HELLO", string(body))
	}

	res.Body.Close()

}

func ExampleHandler() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Name", "golazy")
		w.WriteHeader(200)
		io.Copy(w, r.Body)
	}

	rts := make(chan *RoundTrip, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rt, _ := ServeTo(w, r, handler)
		rts <- rt
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL+"/404", strings.NewReader("hello"))

	server.Client().Do(req)

	rt := <-rts
	fmt.Println(rt.String())

	// Output:
	// URL       : /404
	// Method    : GET
	// Req Header: Accept-Encoding=gzip Content-Length=5 User-Agent=Go-http-client/1.1
	// Req Body  : hello
	// Status    : 200
	// Res Header: X-Name=golazy
	// Res Body  : hello

}
