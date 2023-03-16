package websockets

import (
	"testing"

	"golazy.dev/lazyapp"
	"golazy.dev/lazyapp/apptest"
)

func TestWebsocket(t *testing.T) {

	a := &lazyapp.App{}

	expect := apptest.New(t, a).Expect

}
