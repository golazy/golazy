package layerfs

import (
	"embed"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

//go:embed layer0
var Layer0 embed.FS

//go:embed layer0 layer1 layer2
var Files embed.FS

func TestFs(t *testing.T) {
	FS := New()
	FS.Add(MustDir(Layer0, "layer0"))
	FS.Add(MustDir(Files, "layer1"))
	FS.Add(MustDir(Files, "layer2"))

	assertFileContent := func(path, content string) {
		t.Helper()
		data, err := fs.ReadFile(FS, path)
		if err != nil {
			t.Errorf("expected %s to be a file. Got error: %s", path, err.Error())
			return
		}

		if strings.TrimSpace(string(data)) != content {
			t.Fatalf("expected file %s to have %q, got %q", path, content, string(data))
		}
	}

	assertFileContent("layer0", "layer0")
	assertFileContent("layer1", "layer1")
	assertFileContent("layer2", "layer2")

	assertFileContent("application.html.tpl", "layer1")
	assertFileContent("config/database", "layer2")
}

func TestGlob(t *testing.T) {
	FS := New()
	FS.Add(MustDir(Files, "layer0"))
	FS.Add(MustDir(Files, "layer1"))
	FS.Add(MustDir(Files, "layer2"))

	assertMatch := func(pattern string, files ...string) {
		t.Helper()
		entries, err := fs.Glob(FS, "*/data*")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			t.Logf("entry: %s", entry)
		}

	FILES:
		for _, file := range files {
			for _, entry := range entries {
				if entry == file {
					continue FILES
				}
			}
			t.Errorf("expected pattern %s to find the file %s", pattern, file)
		}
	}

	//assertMatch("*", strings.Split("layer0 layer1 layer2 application.html.tpl config", " ")...)
	assertMatch("config/*", "config/database")
}

func TestFSTest(t *testing.T) {
	FS := New()
	FS.Add(MustDir(Layer0, "layer0"))
	FS.Add(MustDir(Files, "layer1"))
	FS.Add(MustDir(Files, "layer2"))
	err := fstest.TestFS(FS, "application.html.tpl", "config/database", "layer0", "layer1", "layer2")
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenDir(t *testing.T) {
	FS := New()
	FS.Add(MustDir(Layer0, "layer0"))
	FS.Add(MustDir(Files, "layer1"))
	FS.Add(MustDir(Files, "layer2"))

	d, err := FS.Open("config")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	stat, err := d.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if !stat.IsDir() {
		t.Fatal("expected . to be a directory")
	}

	dir, ok := d.(fs.ReadDirFile)
	if !ok {
		t.Fatal("expected . to be a ReadDirFile")
	}

	totalEntries := []fs.DirEntry{}

	for {
		entries, err := dir.ReadDir(1)
		if len(entries) == 0 {
			t.Log("reea entries returned 0 entries and err:", err)
		}
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		if err == io.EOF {
			break
		}
		totalEntries = append(totalEntries, entries...)
		t.Log(entries)
	}
}
