package lazyml

import (
	"bytes"
	"testing"
)

func TestText(t *testing.T) {

	b := &bytes.Buffer{}
	Text("<html lang=\"en\"> &").WriteTo(b)

	if b.String() != "&lt;html lang=&#34;en&#34;&gt; &amp;" {
		t.Error("expectin the text to be escaped Got:", b.String())

	}

}
