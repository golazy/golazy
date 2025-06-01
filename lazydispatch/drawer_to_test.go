package lazydispatch

import (
	"testing"
)

func TestTo(t *testing.T) {

	// h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	//
	//	tests := []struct {
	//		name             string
	//		to               *toSuper
	//		method, path, as string
	//	}{
	//
	//		{"empty", (&Scope{}).To(h), "GET", "/", ""},
	//		{"with_scope", (&Scope{}).Put("").Namespace("admin").Path("secret").As("update_secret").To(h), "PUT", "/secret", "update_secret"},
	//	}
	//
	//	for _, test := range tests {
	//		t.Run(test.name, func(t *testing.T) {
	//			s := test.to.scope
	//			r := s.Routes()[0]
	//			if r.Method != test.method {
	//				t.Errorf("expected %s, got %s", test.method, r.Method)
	//			}
	//			if r.Path != test.path {
	//				t.Errorf("expected %s, got %s", test.path, r.Path)
	//			}
	//			if r.URL != test.path {
	//				t.Errorf("expected %s, got %s", test.path, r.URL)
	//			}
	//			if r.Name != test.as {
	//				t.Errorf("expected %s, got %s", test.as, r.Name)
	//			}
	//
	//		})
	//	}
}
