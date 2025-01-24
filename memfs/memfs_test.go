package memfs

import (
	"bytes"
	"io/fs"
	"testing"
)

func TestMemoryFS(t *testing.T) {

	memfs := New()

	_, err := memfs.Open("/a/b/c/hello.txt")
	if err != fs.ErrNotExist {
		t.Fatal(err)
	}

	memfs.Add("/a/b/c/hello.txt", []byte("hello world"))
	f, err := memfs.Open("/a/b/c/hello.txt")
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	buf.ReadFrom(f)

	if string(buf.String()) != "hello world" {
		t.Fatal("unexpected data")
	}

}

func TestMemFS_Dir(t *testing.T) {
	memfs := New()
	memfs.Add("file", []byte("file"))
	memfs.Add("file/b", []byte("fb"))
	memfs.Add("file/c", []byte("fc"))
	memfs.Add("/c", []byte("c"))

	entries, err := memfs.ReadDir("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatal("unexpected entries")
	}

	entries, err = memfs.ReadDir("file")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatal("unexpected entries")
	}
	for _, e := range entries {
		if e.Name() == "b" {
			if e.IsDir() {
				t.Fatal("unexpected dir")
			}
			info, err := e.Info()
			if err != nil {
				t.Fatal(err)
			}
			if info.Size() != 2 {
				t.Fatal("unexpected size")
			}
			if info.Mode() != fs.ModePerm {
				t.Fatal("unexpected mode")
			}
			if info.Name() != "b" {
				t.Fatal("unexpected name")
			}
			if info.ModTime().IsZero() {
				t.Fatal("unexpected modtime")
			}
			if info.IsDir() {
				t.Fatal("unexpected isdir")
			}

		}
	}

}
