package lazysupport

import "testing"

func TestSet(t *testing.T) {

	set := NewSet("hello", "world")
	if set.Has("moon") {
		t.Fatal("should not have the mooon")
	}
	if !set.Has("world") {
		t.Fatal("it should have the world")
	}
}
