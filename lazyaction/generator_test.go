package lazyaction

import (
	"net/http/httptest"
	"testing"
)

func TestGenerator(t *testing.T) {

	r := &Dispatcher{}

	r.Resource(&GeneratorController{})

	req := httptest.NewRequest("GET", "/generator", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "user" {
		t.Errorf("Expected 'user', got %s", w.Body.String())
	}

}
