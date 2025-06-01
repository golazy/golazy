package apptemplate

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type RunOpts struct {
	Dest string
	Vars map[string]string

	// Logger to use to report changes. If need DefaultLogger will be used
	Logger     Logger
	OnConflict func(string, io.Reader) error
}

type runCtx struct {
	RunOpts
}

func (r *runCtx) write(path string, content io.Reader) error {
	dest := filepath.Join(r.Dest, path)
	_, err := os.Stat(dest)
	if err == nil {

		if r.OnConflict == nil {
			panic(fmt.Errorf("File already exists: %s", path))
		}
		err = r.OnConflict(dest, content)
		if err != nil {
			return fmt.Errorf("Error writing file: %w", err)
		}
		return nil
	}
	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return fmt.Errorf("Error creating directories: %w", err)
	}
	// TODO: create dirs
	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, content)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil

}

func newRun(actions []action, opts RunOpts) error {
	runCtx := runCtx{
		RunOpts: opts,
	}
	if runCtx.OnConflict == nil {
		runCtx.OnConflict = func(path string, content io.Reader) error {
			f, err := os.Open(path)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			e, err := equals(f, content)
			if err != nil {
				panic(err)
			}
			if e {
				return nil
			}
			panic(fmt.Errorf("File already exists: %s", path))
		}
	}
	for _, a := range actions {
		a.Run(runCtx)
	}
	return nil
}

func equals(a, b io.Reader) (bool, error) {
	aData, err := io.ReadAll(a)
	if err != nil {
		panic(err)
	}
	bData, err := io.ReadAll(b)
	return bytes.Equal(aData, bData), nil
}
