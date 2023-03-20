package component

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGFont(t *testing.T) {

}

func TestGFont_Install(t *testing.T) {
	pathPrefix := "install_test"
	defer os.RemoveAll(pathPrefix)
	cachePrefix := "install_cache"
	defer os.RemoveAll(cachePrefix)

	source := &GFont{
		ApiKey: "AIzaSyD8NZhGp0XOhvqZlRGpPxzZ5CnaEychmO0",
		Font:   "Roboto",
	}

	err := source.Install(InstallOptions{
		Path:  pathPrefix,
		Cache: cachePrefix,
	})

	if err != nil {
		t.Fatal(err)
	}

	// Check cache
	s, err := os.Stat(filepath.Join(cachePrefix, "node_modules", "@hotwired", "turbo", "dist", "turbo.es2017-esm.js"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Size() == 0 {
		t.Fatal("File is empty")
	}

	// Check file
	s, err = os.Stat(filepath.Join(pathPrefix, "@hotwired", "turbo", "dist", "turbo.es2017-esm.js"))
	if err != nil {
		t.Fatal(err)
	}
	if s.Size() == 0 {
		t.Fatal("File is empty")
	}
}
