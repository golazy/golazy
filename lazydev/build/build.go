package build

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Options struct {
	Args       []string
	Dir        string
	OutputPath string
	Stdout     io.Writer
	Stderr     io.Writer
}

func Build(bo Options) error {
	err := os.MkdirAll(filepath.Dir(bo.OutputPath), 0755)
	if err != nil {
		return err
	}

	cmd := exec.Command("go", append([]string{"build", "-o", bo.OutputPath}, bo.Args...)...)
	cmd.Stdout = bo.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = bo.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	cmd.Dir = bo.Dir
	return cmd.Run()

}
