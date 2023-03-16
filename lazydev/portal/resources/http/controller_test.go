package http_test

import (
	"portal/apps/portal"
	"testing"

	"golazy.dev/lazyapp/apptest"
)

func TestController(t *testing.T) {

	expect := apptest.New(t, portal.App).Expect

	expect("http://localhost/").Contains("redirect.js")

}
