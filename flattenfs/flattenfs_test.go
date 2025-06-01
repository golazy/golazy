package flattenfs

import (
	"embed"
	"io"
	"testing"

	"github.com/golazy/golazy/lazysupport"
)

//go:embed test1
var FS embed.FS

func TestFlattenFS(t *testing.T) {

	fs := FlattenFS{FS}

	files, err := fs.Glob("**/*.tpl")
	if err != nil {
		t.Fatal(err)
	}

	includes := func(file string, content string) {
		found := false
		for _, f := range files {
			if f == file {
				found = true
				break
			}
		}
		if !found {
			t.Error("file not found", file, "Have", lazysupport.ToSentence("and", files...))
			return
		}
		f, err := fs.Open(file)
		if err != nil {
			t.Error(err)
			return
		}
		data, err := io.ReadAll(f)
		if err != nil {
			t.Error(err)
		}
		if string(data) != content {
			t.Error("Expected", content, "got", string(data))
		}
	}

	includes("test1__1.tpl", "1")
	includes("test1__test2__test3__3.tpl", "3")
	if len(files) != 2 {
		t.Error("Expected 2 files, got", len(files))
	}

}
