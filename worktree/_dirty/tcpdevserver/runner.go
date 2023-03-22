package tcpdevserver

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

//	r.cmd = r.Command()
//	syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL)

type Runner struct {
	Command      func() *exec.Cmd
	AfterRestart func()
	cmd          *exec.Cmd
	i            chan (os.Signal)
	Log          interface{ Print(v ...any) }
}

func (r *Runner) Restart() error {
	return r.Start()
}

func (r *Runner) l(v ...any) {
	if r.Log != nil {
		r.Log.Print(append([]any{"runner:"}, v...)...)
	}
}

func (r *Runner) Start() error {
	r.l("Start()")
	if r.i == nil {
		r.i = make(chan os.Signal, 1)
		signal.Notify(r.i, os.Interrupt)
		go func() {
			<-r.i
			r.l("Got SIGINT")
			r.Stop()
			r.l("Childs are dead")
			os.Exit(0) // This should be centrallyize to allow more than one runner
		}()
	}
	if r.cmd != nil && r.cmd.Process != nil {
		if err := r.cmd.Process.Signal(syscall.Signal(0)); err == nil {
			r.l("Start: Process runing. Calling Stop()")
			r.Stop()
		}
	}
	r.cmd = r.Command()
	r.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	r.l("Start: Starting process:", r.cmd.String())

	err := r.cmd.Start()
	if err != nil {
		return err
	}

	if r.AfterRestart != nil {
		r.l("Notifying AfterRestart")
		r.AfterRestart()
	}

	return nil
}

// Stop tries to kill the process group and waits for the command to finish.
// If the process is not running it does not return any error
func (r *Runner) Stop() {
	if r.cmd == nil || r.cmd.Process == nil {
		return
	}
	r.l("Stop: Killing process", r.cmd.Process.Pid)
	syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL)
	r.cmd.Process.Wait()
}
