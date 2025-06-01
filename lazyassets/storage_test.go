package lazyassets

import (
	"embed"
	"testing"
)

//go:embed test_assets/*
var TestAssetsFS embed.FS

func TestWithoutHash(t *testing.T) {

	_, _, err := withoutHash("hello.world")
	if err != errNoHash {
		t.Errorf("Expected %q. Got: %q", errNoHash, err)
	}

	p, hash, err := withoutHash("asdf-123asdf.js")
	if err != nil {
		t.Error(err)
	}
	if hash != "123asdf" {
		t.Errorf("Expected %q. Got: %q", "123asdf", hash)
	}

	if p != "asdf.js" {
		t.Errorf("Expected %q. Got: %q", "asdf.js", p)
	}

	// With path
	p, hash, err = withoutHash("/js/asdf-zxcv.js")
	if err != nil {
		t.Error(err)
	}
	if hash != "zxcv" {
		t.Errorf("Expected %q. Got: %q", "zxcv", hash)
	}
	if p != "/js/asdf.js" {
		t.Errorf("Expected %q. Got: %q", "/js/asdf.js", p)
	}

	// No extension
	p, hash, err = withoutHash("/data/asdf-123")
	if err != nil {
		t.Error(err)
	}
	if p != "/data/asdf" {
		t.Errorf("Expected %q. Got: %q", "/data/asdf", p)
	}
	if hash != "123" {
		t.Errorf("Expected %q. Got: %q", "123", hash)
	}

	// Ending in underscore
	_, _, err = withoutHash("/data/asdf-")
	if err != errNoHash {
		t.Error(err)
	}

}

func TestManager_Find(t *testing.T) {
	ts := NewServerTest(t)
	ts.AddFS(TestAssetsFS, "test_assets")

	f := ts.Find("missing")
	if f != nil {
		t.Error("Expected nil. Got:", f)
	}

	f = ts.Find("/@test/hello.world")
	if f == nil {
		t.Fatal("Expected file. Got:", f)
	}

}

func TestManager_Permalinkize(t *testing.T) {
	s := (&Server{Storage: &Storage{}})
	s.AddFS(TestAssetsFS, "test_assets")
	s.AddFile("/a.png", []byte(""))

	expect := func(input, expectation string) {
		t.Helper()
		out := formatCSSPermalink(s, input)
		if out != expectation {
			t.Errorf("Expected \n%q to produce \n%q. Got: \n%q", input, expectation, out)
		}
	}

	expect("", "")
	expect("hola", "hola")
	expect("123 url('hola') 456 url('hey') 789", "123 url('hola') 456 url('hey') 789")
	expect("123 url('/a.png') 456 url('hey') 789", "123 url('/a-e3b0c44298fc.png') 456 url('hey') 789")
}
