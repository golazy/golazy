package devserver

import (
	"os"
	"os/exec"
	"syscall"
)

type runOpts struct {
	Dir  string
	Path string
	Env  []string
}

type cWriter chan ([]byte)

func (c cWriter) Write(args []byte) (int, error) {
	c <- args
	return len(args), nil
}

func run(opts runOpts) (stdout, stderr <-chan ([]byte), exit <-chan (int), kill func(), err error) {

	stdC := make(chan []byte)
	errC := make(chan []byte)
	exitC := make(chan (int))

	cmd := exec.Command(opts.Path)
	cmd.Dir = opts.Dir
	cmd.Stdout = cWriter(stdC)
	cmd.Stderr = cWriter(errC)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, opts.Env...)
	err = cmd.Start()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	go func() {
		cmd.Wait()
		exitC <- cmd.ProcessState.ExitCode()
		close(exitC)
		close(stdC)
		close(errC)
	}()

	return stdC,
		errC,
		exitC,
		func() { syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) },
		nil
}
