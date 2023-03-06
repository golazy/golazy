package component

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNpm_Install(t *testing.T) {
	pathPrefix := "install_test"
	defer os.RemoveAll(pathPrefix)
	cachePrefix := "install_cache"
	defer os.RemoveAll(cachePrefix)

	source := &Npm{
		Name:    "@hotwired/turbo",
		Version: "7.2.5",
		Imports: map[string]string{"turbo": "dist/turbo.es2017-esm.js"},
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

func TestNpm_Uninstall(t *testing.T) {
	pathPrefix := "install_test"
	defer os.RemoveAll(pathPrefix)
	cachePrefix := "install_cache"
	defer os.RemoveAll(cachePrefix)

	source := &Npm{
		Name:    "@hotwired/turbo",
		Version: "7.2.5",
		Imports: map[string]string{"turbo": "dist/turbo.es2017-esm.js"},
	}

	opts := InstallOptions{
		Path:  pathPrefix,
		Cache: cachePrefix,
	}

	err := source.Install(opts)
	if err != nil {
		t.Fatal(err)
	}

	err = source.Uninstall(opts)
	if err != nil {
		t.Fatal(err)
	}

	// Check cache
	f := filepath.Join(cachePrefix, "node_modules", "@hotwired", "turbo", "dist", "turbo.es2017-esm.js")
	_, err = os.Stat(f)
	if err == nil {
		t.Error("File exists", f)
	}

	// Check file
	f = filepath.Join(pathPrefix, "@hotwired", "turbo", "dist", "turbo.es2017-esm.js")
	_, err = os.Stat(f)
	if err == nil {
		t.Error("File exists", f)
	}
}

func TestNpm_Installed(t *testing.T) {
	pathPrefix := "install_test"
	defer os.RemoveAll(pathPrefix)
	cachePrefix := "install_cache"
	defer os.RemoveAll(cachePrefix)

	source := &Npm{
		Name:    "@hotwired/turbo",
		Version: "7.2.5",
		Imports: map[string]string{"turbo": "dist/turbo.es2017-esm.js"},
	}

	opts := InstallOptions{
		Path:  pathPrefix,
		Cache: cachePrefix,
	}

	if source.Installed(opts) {
		t.Error("Should not be installed")
	}

	err := source.Install(opts)
	if err != nil {
		t.Fatal(err)
	}

	if !source.Installed(opts) {
		t.Error("Should be installed")
	}
}

func TestNpm_ImportMap(t *testing.T) {
	source := &Npm{
		Name:    "@hotwired/turbo",
		Version: "7.2.5",
		Imports: map[string]string{"turbo": "dist/turbo.es2017-esm.js"},
	}

	source.ImportMap()
	reflect.DeepEqual(source.Imports, map[string]string{"turbo": "dist/turbo.es2017-esm.js"})

}
