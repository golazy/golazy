package lazyaction

import (
	"net/http/httptest"
	"testing"

	"golazy.dev/lazyview/nodes"
)

func TestLayout(t *testing.T) {

	nodes.Beautify = false
	r := &Dispatcher{}

	r.Resource(&LayoutController{})

	req := httptest.NewRequest("GET", "/layout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	expected := `<!DOCTYPE html><html><head><body>index`
	if w.Body.String() != expected {
		t.Errorf("Expected %q, got %s", expected, w.Body.String())
	}

}

func TestLayout_Embebed(t *testing.T) {

	r := &Dispatcher{}

	r.Resource(&PagesWithLayout{}, &ResourceOptions{Path: "layout"})

	req := httptest.NewRequest("GET", "/layout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "--embebed index--" {
		t.Errorf("Expected --embebed index-- , got %s", w.Body.String())
	}

}
