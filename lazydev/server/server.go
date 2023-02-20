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
	"time"

	"golazy.dev/lazydev/filewatcher"
	"golazy.dev/lazydev/server/internal/hrouter"
)

type nonCloseListener struct {
	net.Listener
}

type BuildError error

func (l nonCloseListener) Close() error { return nil }

type Server struct {
	BuildArgs []string
	BuildDir  string
	Addr      string
	Err       error
	// HttpHandler handles all non-https requests
	HttpHandler http.Handler
	// FallbackHandler handles all https requests when the server is not running
	FallbackHandler http.Handler
	// PrefixHandler handles all requests with the given prefix
	PrefixHandler http.Handler
	Prefix        string
	router        *hrouter.Server
	buildFile     string
	child         *os.Process
	childL        net.TCPListener
	buildOutput   *bytes.Buffer
}

type action func() action

func (s *Server) ListenAndServe() error {
	var current action
	s.buildOutput = &bytes.Buffer{}

	s.router = &hrouter.Server{
		HTTPHandler:     s.HttpHandler,
		PrefixHandler:   s.PrefixHandler,
		Prefix:          "/golazy",
		FallbackHandler: s.FallbackHandler,
		Addr:            s.Addr,
	}

	go func() {
		// Main loop
		for current = s.build; current != nil; {
			current = current()
		}

	}()

	s.router.ListenAndServe()

	if s.buildFile != "" {
		os.Remove(s.buildFile)
	}

	return s.Err
}

func (s *Server) build() action {
	s.router.CBClose()
	fmt.Println("> Building...")

	// Create the tempfile
	temp, err := os.CreateTemp("", "lazydev")
	if err != nil {
		s.Err = fmt.Errorf("can't create temp file: %w", err)
		return nil
	}
	s.buildFile = temp.Name()
	temp.Close()

	// Run the build
	cmd := exec.Command("go", append([]string{"build", "-o", temp.Name()}, s.BuildArgs...)...)
	s.buildOutput.Reset()
	cmd.Stdout = s.buildOutput
	cmd.Stderr = s.buildOutput
	cmd.Dir = s.BuildDir
	err = cmd.Run()
	if err != nil {
		fmt.Println(prefix("|", s.buildOutput.String()))
		s.Err = BuildError(fmt.Errorf("error building: %w", err))
		return s.error
	}

	return s.start
}

func prefix(prefix, s string) string {
	out := ""
	for _, line := range strings.Split(s, "\n") {
		out += prefix + line + "\n"
	}
	return out
}

func (s *Server) start() action {
	fmt.Println("> Starting...")
	// Start the child

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.Err = fmt.Errorf("can't listen: %w", err)
		return s.error
	}

	s.childL = *l.(*net.TCPListener)

	cmd := exec.Command(s.buildFile, "--port", "fd:3")
	file, err := s.childL.File()
	if err != nil {
		s.Err = fmt.Errorf("BUG: can't get file from listener: %w", err)
		return nil
	}

	cmd.Dir = s.BuildDir
	cmd.ExtraFiles = []*os.File{file}

	logger := log.New(os.Stdout, "||>  ", 0)
	logW := logWriter{logger}
	cmd.Stdout = logW
	cmd.Stderr = logW
	err = cmd.Start()
	if err != nil {
		s.childL.Close()
		s.Err = fmt.Errorf("can't start child process: %w", err)
		return s.error
	}

	s.child = cmd.Process

	return s.ready
}

func (s *Server) ready() action {
	fmt.Println("> Ready! (TODO: wait for interrupt)")
	addr := s.childL.Addr().String()
	childUrl, _ := url.Parse("http://" + addr)
	s.router.CBOpen(childUrl)

	// Wait for the child to die
	running := make(chan struct{})
	go func() {
		defer close(running)
		childState, err := s.child.Wait()
		if err != nil {
			s.Err = fmt.Errorf("error waiting for child process: %w", err)
			return
		}
		s.Err = fmt.Errorf("child process exited with status %d", childState.ExitCode())

	}()

	// Ensure we kill the child before next step
	defer func() {
		s.child.Kill()
		<-running
	}()

	changes, close, err := watch(s.BuildDir)
	if err != nil {
		<-running
		// TODO: handle unexpected quit
		return s.error
	}

	defer close()

	select {
	case change := <-changes:
		fmt.Println("> Change detected, rebuilding...", change)
		return s.build
	case <-running:
		fmt.Println("> Child process exited, rebuilding...")
		return s.error
	}

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

func (s *Server) error() action {
	fmt.Println("Err:", s.Err)
	// error
	s.Err = nil

	changes, close, err := watch(s.BuildDir)
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
