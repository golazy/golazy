package app

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

type buildOptions struct {
	Dir     string
	TempDir string
}

type buildResult struct {
	Err  error
	Out  []byte
	Path string
}

type BuildError buildResult

func (b BuildError) Error() string {
	return b.Err.Error()
}

func (b BuildError) Unwrap() error {
	return b.Err
}

func build(bo *buildOptions) (br *buildResult) {
	br = new(buildResult)

	br.Path = filepath.Join(bo.TempDir, fmt.Sprintf("lazyapp-%d", time.Now().UnixMilli()))

	cmd := exec.Command("go", "build", "-o", br.Path)
	cmd.Dir = bo.Dir

	b := &bytes.Buffer{}
	cmd.Stdout = b
	cmd.Stderr = b

	br.Err = cmd.Run()
	br.Out = b.Bytes()

	if br.Err != nil {
		return
	}

	return br
}
