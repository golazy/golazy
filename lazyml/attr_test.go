package lazyml

import "testing"

func TestAttrWiteTo(t *testing.T) {

	test := func(expectation string, key string, value ...string) {
		t.Helper()
		attr := NewAttr(key, value...)
		if attr.String() != expectation {
			t.Errorf("Expected: %q Got %q", expectation, attr.String())
		}
	}
	test("download", "download")

	test("", "")

	test("href", "href")
	test("href=\"\"", "href", "")
	test("href=http://google.com", "href", "http://google.com")
	test("style=width:30px", "style", "width:30px")

}
