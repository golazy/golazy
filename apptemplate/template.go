package apptemplate

import (
	"io/fs"
)

type Template struct {
	Name    string
	actions []action
}

func (t *Template) Run(opts RunOpts) error {
	if opts.Logger == nil {
		opts.Logger = DefaultLogger
	}
	if opts.Vars == nil {
		opts.Vars = make(map[string]string)
	}
	return newRun(t.actions, opts)
}

type action interface {
	Name() string
	Steps() int
	Run(runCtx)
}
type basicAction struct {
	name string
	Fn   func(runCtx)
}

func (a basicAction) Name() string {
	return a.name
}
func (a basicAction) Steps() int {
	return 1
}

func (a basicAction) Run(ctx runCtx) {
	a.Fn(ctx)
}

func (t *Template) Copy(source fs.FS) *Template {
	t.actions = append(t.actions, basicAction{
		name: "Copy",
		Fn: func(ctx runCtx) {
			eachFile := func(path string, entry fs.DirEntry, err error) error {
				if entry.IsDir() {
					return nil
				}

				file, err := source.Open(path)
				if err != nil {
					panic(err.Error())
				}
				err = ctx.write(path, file)
				if err != nil {
					panic(err.Error())
				}
				err = file.Close()
				if err != nil {
					panic(err.Error())
				}
				return nil
			}

			err := fs.WalkDir(source, ".", eachFile)
			if err != nil {
				panic(err.Error())
			}

		},
	})
	return t
}
