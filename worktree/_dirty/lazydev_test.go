package lazydev

import (
	"os"
	"testing"
)

func TestLazyDev(t *testing.T) {

	err := os.Chdir("test_app")
	if err != nil {
		t.Fatal(err)
	}

	err = Serve(nil)
	if err != nil {
		t.Fatal(err)
	}

}
