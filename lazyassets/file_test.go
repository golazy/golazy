package lazyassets

import "testing"

func TestFile(t *testing.T) {

	f := newStaticFile("/asdf.css", "test:1", []byte("hola"))
	if f.Path() != "/asdf.css" {
		t.Error("Expected /asdf.css. Got:", f.Path())
	}

	if f.Permalink() != "/asdf-b221d9dbb083.css" {
		t.Error("Expected /asdf-52d61becd550.css. Got:", f.Permalink())
	}

	if f.Hash() != "b221d9dbb083" {
		t.Error("Expected 52d61becd550. Got:", f.Hash())
	}

	if f.MimeType() != "text/css; charset=utf-8" {
		t.Error("Expected text/css. Got:", f.MimeType())
	}

}
