package devserver

import (
	"bytes"
	"fmt"
	"os"

	lazydev_build "golazy.dev/lazydev/build"
)

type buildOpts struct {
	Dir  string
	Args []string
}

func build(opts buildOpts) (file string, output []byte, err error) {

	// Create the tempfile
	temp, err := os.CreateTemp("", "lazydev")
	if err != nil {
		err = fmt.Errorf("can't create temp file: %w", err)
		return
	}
	file = temp.Name()
	temp.Close()

	buf := &bytes.Buffer{}

	err = lazydev_build.Build(lazydev_build.Options{
		Dir:        opts.Dir,
		Args:       opts.Args,
		OutputPath: file,
		Stdout:     buf,
		Stderr:     buf,
	})

	output = buf.Bytes()
	return
}
