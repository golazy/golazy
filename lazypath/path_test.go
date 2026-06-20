package lazypath

import "testing"

func TestAppendURLParams(t *testing.T) {
	path := AppendURLParams("/polls/123/admin", URLParams{
		"token": "secret token",
		"empty": nil,
	})
	if path != "/polls/123/admin?token=secret+token" {
		t.Fatalf("path = %q, want query params", path)
	}
}

func TestAppendURLParamsKeepsExistingQuery(t *testing.T) {
	path := AppendURLParams("/polls?tab=admin", URLParams{"token": "secret"})
	if path != "/polls?tab=admin&token=secret" {
		t.Fatalf("path = %q, want appended query params", path)
	}
}

func TestAppendURLParamsKeepsFragmentLast(t *testing.T) {
	path := AppendURLParams("/polls#admin", URLParams{"token": "secret"})
	if path != "/polls?token=secret#admin" {
		t.Fatalf("path = %q, want query params before fragment", path)
	}
}
