package lazydispatch

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestParamsController struct {
}

func (*TestParamsController) GETRequest(r *http.Request) error {
	if r == nil {
		return errors.New("Request is nil")
	}
	return nil
}
func (*TestParamsController) GETResponse(w http.ResponseWriter) error {
	if w == nil {
		return errors.New("Request is nil")
	}
	return nil
}
func (*TestParamsController) GETRoute(r *Route) error {
	if r == nil {
		return errors.New("Request is nil")
	}
	return nil
}

func TestParam(t *testing.T) {

	d := New()
	d.Draw(func(s *Scope) {
		s.Resources(&TestParamsController{})
	})

	test := func(path string) {
		t.Helper()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", path, nil)
		d.ServeHTTP(rec, req)
		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Errorf("Expected status code 200, got %d", res.StatusCode)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "" {
			t.Errorf("Expected empty body, got %s", string(body))
		}
	}

	test("/test_params/request")
	test("/test_params/response")
	test("/test_params/route")

}
