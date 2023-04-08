package lazyaction_test

import (
	"testing"

	"golazy.dev/lazyaction"
	"golazy.dev/lazyapp/apptest"
)

func TestRedirects(t *testing.T) {

	r := &lazyaction.Dispatcher{}

	r.Resource(&lazyaction.RedirectController{})

	expect := apptest.New(t, r).Expect

	expect("/redirect/one").Code(301).Location("/one")
	expect("/redirect/two").Code(301).Location("/two")
	expect("/redirect/three").Code(307).Location("/three")
	expect("/redirect/four").Code(307).Location("/four")
	expect("/redirect/five").Code(307).Location("/five")

}
