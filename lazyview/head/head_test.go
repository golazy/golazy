package head

import (
	"bytes"
	"io"
	"testing"

	"golang.org/x/exp/rand"
)

type fake struct {
	t Category
}

func (f fake) Category() Category {
	return f.t
}
func (f fake) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(f.t.String()))
	return int64(n), err
}

func TestHeadOrder(t *testing.T) {

	h := &Head{}
	for _, i := range rand.Perm(8) {
		h.Add(fake{Category(i)})
	}

	b := &bytes.Buffer{}
	h.WriteTo(b)
	if b.String() != "TitleBaseLinkMetaStyleScriptNoscriptTemplate" {
		t.Errorf("Expected \n%q but got \n%q", "HeadTitleHeadBaseHeadLinkHeadMetaHeadStyleHeadScriptHeadNoscriptHeadTemplate", b.String())
	}

}
