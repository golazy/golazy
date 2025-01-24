package apptemplate

import (
	"testing"
	"testing/fstest"
)

func TestMemFS(t *testing.T) {

	mfs := MemFS{
		"hello.txt":                "Hello, World!",
		"world.txt":                "World, Hello!",
		"dir/hellodir.txt":         "Hello, World!",
		"dir2/dir3/dir4/hello.txt": "Hello, World!",
		"dir2/dir3/dir4/bye.txt":   "Hello, World!",
		"dir2/gopher.con":          "Hello, World!",
	}
	err := fstest.TestFS(mfs,
		"hello.txt", "world.txt", "dir/hellodir.txt", "dir2/dir3/dir4/hello.txt",
		"dir2/dir3/dir4/bye.txt", "dir2/gopher.con")
	if err != nil {
		t.Fatal(err)
	}

}
