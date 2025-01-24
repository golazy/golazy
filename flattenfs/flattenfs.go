package flattenfs

import (
	"io/fs"
	"strings"
)

type FlattenFS struct {
	fs.FS
}

// Flat replaces all "/" with FlatSeparator in the pattern.
func Flat(pattern string) string {
	return strings.ReplaceAll(pattern, "/", FlatSeparator)
}

// UnFlat replaces all FlatSeparator with "/" in the pattern.
func UnFlat(pattern string) string {
	return strings.ReplaceAll(pattern, FlatSeparator, "/")
}

var FlatSeparator = "__"

func (f FlattenFS) Glob(pattern string) ([]string, error) {

	files, err := Glob(f.FS, pattern)
	if err != nil {
		return nil, err
	}

	for i, file := range files {
		files[i] = strings.ReplaceAll(file, "/", FlatSeparator)
	}
	return files, nil
}

func (f FlattenFS) Open(name string) (fs.File, error) {

	path := strings.ReplaceAll(name, FlatSeparator, "/")
	return f.FS.Open(path)

}
