package lazysupport

import "testing"

func TestToSentence(t *testing.T) {

	test := func(expectation, last_join string, s ...string) {
		t.Helper()
		out := ToSentence(last_join, s...)
		if out != expectation {
			t.Errorf("Expecting %q Got %q", expectation, out)
		}
	}

	test("", "")
	test("uno", "", "uno")
	test("uno y dos", "y", "uno", "dos")
	test("one, two and three", "", "one", "two", "three")

}
