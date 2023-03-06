package portal

import (
	"testing"

	"golazy.dev/lazyapp/apptest"
)

func TestApp(t *testing.T) {

	a := App

	expect := apptest.New(t, a).Expect

	expect("/").
		Code(200).
		Contains("hellos")
}
