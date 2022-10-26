package lazydev

import (
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golazy.dev/lazydev/filewatcher"
	"golazy.dev/lazydev/runner"
)

func (s *server) startParent(h http.Handler) error {
	// listen Addr
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = s.HTTPSAddr
	} else {
		if !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return err
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	// Child runner
	cmd, err := childCmd(l)
	if err != nil {
		return err
	}
	r := runner.New(cmd, nil)

	running := false
	err = r.Start()
	if err != nil {
		return err
	}
	defer func() {
		if running {
			r.Stop()
		}
	}()

	// File monitoring
	fw, err := filewatcher.New("")
	if err != nil {
		return err
	}
	changeSet, err := fw.Watch()
	if err != nil {
		return err
	}
	defer func() {
		fw.Close()
	}()

	// Signal handling
	intSignal := make(chan os.Signal, 1)
	signal.Notify(intSignal, os.Interrupt)

	// Event handling
	for {
		select {
		case event := <-r.Events:
			if _, ok := event.(runner.EventStarted); ok {
				running = true
			}
			if _, ok := event.(runner.EventStopped); ok {
				running = false
			}
		case cs := <-changeSet:
			log.Printf("%T %+v", cs, cs)
			err := r.Restart()
			if err != nil {
				log.Println("Error while restarting:", err)
			}

		case <-intSignal:
			if running == false {
				return nil
			}
			r.Stop()
			wait := time.After(time.Second * 2)
			intCount := 1
			for wait != nil {
				select {
				case <-wait:
					wait = nil
				case event := <-r.Events:
					log.Printf("%T %+v", event, event)
					if _, ok := event.(runner.EventStopped); ok {
						running = false
						return nil
					}
				case <-intSignal:
					if intCount == 1 {
						log.Println("Killing the process")
						intCount += 1
						r.Signal(syscall.SIGKILL)
						continue
					}
					return nil
				}
			}
			return nil
		}
	}
}

func childCmd(l *net.TCPListener) (*exec.Cmd, error) {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	p := "*.go"
	if wd != "" {
		p = filepath.Join(wd, p)
	}
	files, err := filepath.Glob(p)
	if err != nil {
		log.Fatal(err)
	}

	args := []string{"run"}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		args = append(args, file)
	}

	file, err := l.File()
	if err != nil {
		return nil, err
	}
	log.Println("Listener is", file)
	cmd := exec.Command("go", args...)
	cmd.ExtraFiles = []*os.File{file}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, childEnvKey+"=true")

	logger := log.New(os.Stdout, "child:  ", 0)
	logW := logWriter{logger}
	cmd.Stdout = logW
	cmd.Stderr = logW

	return cmd, nil
}

type logWriter struct {
	*log.Logger
}

func (l logWriter) Write(args []byte) (int, error) {
	l.Println(string(args))
	return len(args), nil
}
