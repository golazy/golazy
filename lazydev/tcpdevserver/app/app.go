package app

import (
	"os"
	"os/exec"
	"sync"
	"time"
)

type GoAppOptions struct {
	ExtraFiles []*os.File
	Env        []string
	Dir        string
}

func New(opt GoAppOptions) *GoApp {
	tempDir, err := os.MkdirTemp("", "goapp-*")
	if err != nil {
		panic(err)
	}

	app := &GoApp{
		extraFiles: opt.ExtraFiles,
		env:        opt.Env,
		dir:        opt.Dir,
		tempDir:    tempDir,
		Events:     make(chan (Event)),
		stdout:     make(chan ([]byte)),
		stderr:     make(chan ([]byte)),
	}

	// Process stdout and stdin
	go func() {
		for {
			select {
			case data, ok := <-app.stdout:
				if !ok {
					for {
						data, ok := <-app.stderr
						if !ok {
							return
						}
						app.Events <- EventAppStderr{newEvent("app_stderr"), data}
					}
				}
				app.Events <- EventAppStdout{newEvent("app_stdout"), data}
			case data, ok := <-app.stderr:
				if !ok {
					for {
						data, ok := <-app.stdout
						if !ok {
							return
						}
						app.Events <- EventAppStdout{newEvent("app_stdout"), data}
					}
				}
				app.Events <- EventAppStderr{newEvent("app_stderr"), data}
			}
		}
	}()

	return app

}

type GoApp struct {
	l          sync.Mutex
	extraFiles []*os.File
	env        []string
	dir        string
	tempDir    string
	Events     chan (Event)
	cmd        *Cmd
	stdout     chan ([]byte)
	stderr     chan ([]byte)
}

func (app *GoApp) Start() {
	app.l.Lock()
	defer app.l.Unlock()
	// build the app
	app.Events <- EventAppBuildStart{
		EventBase: newEvent("build_start"),
	}

	br := build(&buildOptions{
		Dir:     app.dir,
		TempDir: app.tempDir,
	})

	app.Events <- EventAppBuildFinish{
		EventBase: newEvent("build_finish"),
		Err:       br.Err,
		Out:       br.Out,
	}

	if br.Err != nil {
		app.Events <- EventAppBuildFailure{
			EventBase: newEvent("build_failure"),
			Err:       br.Err,
			Out:       br.Out,
		}
		return
	} else {
		app.Events <- EventAppBuildSuccess{
			newEvent("build_success"),
		}
	}

	if app.cmd != nil {
		app.Events <- EventAppStopping{EventBase: newEvent("app_stopping"), Pid: app.cmd.Process.Pid}
		app.cmd.Stop()
	}

	app.cmd = &Cmd{
		Cmd: exec.Cmd{
			Env:        app.env,
			ExtraFiles: app.extraFiles,
			Path:       br.Path,
			Dir:        app.dir,
			Stdout:     chanWriter(app.stdout),
			Stderr:     chanWriter(app.stderr),
		},
		WaitTime: time.Second,
	}

	err := app.cmd.Start()
	if err != nil {
		app.Events <- EventAppStartFail{newEvent("app_start_fail"), err}
		return
	}
	app.Events <- EventAppStart{
		EventBase: newEvent("app_start"),
		Pid:       app.cmd.Process.Pid,
	}
	go func() {
		<-app.cmd.WaitCh
		app.Events <- EventAppStop{EventBase: newEvent("app_stop")}
	}()
}

func (app *GoApp) Stop() {
	app.Events <- EventAppStopping{EventBase: newEvent("app_stopping"), Pid: app.cmd.Process.Pid}
	app.cmd.Stop()
}

func (app *GoApp) Clean() {
	os.RemoveAll(app.tempDir)
}
