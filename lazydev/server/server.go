package server

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/adrg/xdg"
	"golazy.dev/lazydev/autocerts"
	"golazy.dev/lazydev/build"
	"golazy.dev/lazydev/filewatcher"
	"golazy.dev/lazydev/server/multihttp"
)

type BuildError error

type DevApp interface {
	http.Handler
	Event(Event)
}

type Options struct {
	App       DevApp
	Addr      string
	BuildArgs []string
	BuildDir  string
}

func New(opts Options) *Server {
	if opts.App == nil {
		panic("no app set")
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:2000"
	}
	if opts.BuildDir == "" {
		panic("no build dir set")
	}

	file, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		file = "golazy.pem"
	}

	tls, err := autocerts.TLSConfigFile(file)
	if err != nil {
		panic(err)
	}

	return &Server{
		s: &multihttp.Server{
			Handler:   opts.App,
			Addr:      opts.Addr,
			TLSConfig: tls,
		},
		opts:        opts,
		buildOutput: &bytes.Buffer{},
	}
}

type Server struct {
	s           *multihttp.Server
	opts        Options
	err         error
	buildFile   string
	child       *os.Process
	childW      chan (dieMsg)
	buildOutput *bytes.Buffer
}

type action func() action

func (s *Server) ListenAndServe() error {
	var current action

	go func() {
		for current = s.build; current != nil; {
			current = current()
		}

	}()
	s.notify(EventListen{})

	defer func() {
		if s.buildFile != "" {
			os.Remove(s.buildFile)
		}
	}()

	return s.s.ListenAndServe()
}

func (s *Server) notify(e Event) {
	s.opts.App.Event(e)
}

func (s *Server) build() action {
	s.notify(EventBuildStart{})

	// Create the tempfile
	temp, err := os.CreateTemp("", "lazydev")
	if err != nil {
		s.err = fmt.Errorf("can't create temp file: %w", err)
		return nil
	}
	s.buildFile = temp.Name()
	temp.Close()

	s.buildOutput.Reset()

	err = build.Build(build.Options{
		Dir:        s.opts.BuildDir,
		Args:       s.opts.BuildArgs,
		OutputPath: s.buildFile,
		Stdout:     s.buildOutput,
		Stderr:     s.buildOutput,
	})
	if err != nil {
		s.notify(EventBuildError{Out: s.buildOutput.Bytes()})
		fmt.Println(prefix("|", s.buildOutput.String()))
		s.err = BuildError(fmt.Errorf("error building: %w", err))
		return s.error
	}

	s.notify(EventBuildSuccess{})

	return s.start
}

func prefix(prefix, s string) string {
	out := ""
	for _, line := range strings.Split(s, "\n") {
		out += prefix + line + "\n"
	}
	return out
}

func getFreePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	parts := strings.Split(l.Addr().String(), ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid address: %s", l.Addr().String())
	}
	return parts[1], nil
}

func (s *Server) start() action {
	fmt.Println("> Starting...")
	// Start the child

	cmd := exec.Command(s.buildFile)
	cmd.Dir = s.opts.BuildDir
	logger := log.New(os.Stdout, "||>  ", 0)
	logW := logWriter{logger}
	cmd.Stdout = logW
	cmd.Stderr = logW
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	port, err := getFreePort()
	if err != nil {
		panic(err)
	}
	cmd.Env = append(os.Environ(), "PORT="+port)

	backendURL, err := url.Parse("http://127.0.0.1:" + port)
	if err != nil {
		panic(err)
	}

	err = cmd.Start()
	if err != nil {
		s.notify(EventAppStartError{Err: err})
		s.err = err
		return s.error
	}

	s.child = cmd.Process

	s.childW = make(chan dieMsg)
	go func() {
		childState, err := s.child.Wait()
		s.childW <- dieMsg{childState, err}
	}()

	s.notify(EventAppStart{backendURL})
	return s.ready
}

type dieMsg struct {
	ps  *os.ProcessState
	err error
}

func (s *Server) kill() action {
	syscall.Kill(-s.child.Pid, syscall.SIGKILL)
	<-s.childW
	s.notify(EventAppStop{Reason: "killed", Expected: true})
	return s.build
}

func (s *Server) ready() action {
	fmt.Println("> Ready! (TODO: wait for interrupt)")

	// Ensure we kill the child before next step
	changes, close, err := watch(s.opts.BuildDir)
	if err != nil {
		panic(err)
	}
	defer close()

	select {
	case die := <-s.childW:
		reason := fmt.Sprintf("app returned %d", die.ps.ExitCode())
		s.notify(EventAppStop{Reason: reason, Expected: false})
		return s.build
	case change := <-changes:
		s.notify(EventFSChange{&change})
	}
	return s.kill
}

func (s *Server) error() action {
	fmt.Println("Err:", s.err)
	// error
	s.err = nil

	changes, close, err := watch(s.opts.BuildDir)
	if err != nil {
		time.Sleep(time.Second)
		return s.build
	}
	defer close()

	fmt.Println("Waiting for changes...")
	<-changes

	return s.build
}

type logWriter struct {
	*log.Logger
}

func (l logWriter) Write(args []byte) (int, error) {
	l.Println(string(args))
	return len(args), nil
}

func watch(path string) (changes <-chan (filewatcher.ChangeSet), close func(), err error) {

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, nil, fmt.Errorf("can't get absolute path: %w.\nKilling and building again", err)
	}

	fw, err := filewatcher.New(abs)
	if err != nil {
		return nil, nil, fmt.Errorf("can't start the file watcher: %w.\nKilling and building again", err)
	}
	changes, err = fw.Watch()
	if err != nil {
		return nil, nil, fmt.Errorf("can't watch directory: %w.\nKilling and building again", err)
	}
	close = func() { fw.Close() }
	return

}
