package head

import (
	"bytes"
	"testing"
)

type tester struct {
	*testing.T
}

func (t *tester) expect(s Script, out string) {
	t.Helper()
	b := &bytes.Buffer{}
	s.WriteTo(b)
	if b.String() != out {
		t.Errorf("Expected \n%q but got \n%q", out, b.String())
	}
}
