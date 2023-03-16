package portal

import (
	"testing"

	"golazy.dev/lazyapp/apptest"
)

func TestApp(t *testing.T) {

	expect := apptest.New(t, App).Expect

	expect("/")
	expect("/")
	expect("/")
	expect("/").
		Code(200).
		Contains("hola")

}
