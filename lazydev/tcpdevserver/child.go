package tcpdevserver

/*
import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type Rusnner struct {
	sync.Mutex
	fd      *os.File
	l       *net.TCPListener
	cmd     *exec.Cmd
	id      int
	dir     string
	running bool
}

func (r Runner) prepareCmd() error {
	glob := "*.go"
	if r.dir != "" {
		glob = filepath.Join(r.dir, glob)
	}

	files, err := filepath.Glob(glob)
	if err != nil {
		return err
	}

	args := []string{"run"}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		args = append(args, file)
	}

	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", envName, r.id))
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.ExtraFiles = []*os.File{r.fd}
	r.cmd = cmd
	return nil
}

func (r Runner) Start() error {
	r.Lock()
	if r.running {
		return fmt.Errorf("Child already running")
	}

	err := r.prepareCmd()
	if err != nil {
		return err
	}
	r.running = true
	r.Unlock()
	defer func() {
		r.Lock()
		r.running = false
		r.Unlock()
	}()
	return r.cmd.Run()
}

func (r Runner) Stop() error {
	if !r.running {
		return nil
	}
	syscall.Kill(-r.cmd.Process.Pid, syscall.SIGKILL)
	return nil
}

func (r Runner) Restart() error {
}

*/
