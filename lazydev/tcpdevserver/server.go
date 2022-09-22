package tcpdevserver

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var counter int

const envName = "TCPDEVSERVER"

type Server struct {
	Addr        string
	Dir         string
	Child       func(l net.Listener) error
	AfterListen func()
	r           Runner
	l           *net.TCPListener
	fd          *os.File
	id          int
	Log         *log.Logger
}

func (s *Server) log(v ...interface{}) {
	if s.Log != nil {
		if s.isParent() {
			s.Log.Print(append([]interface{}{"Server(Parent): "}, v...)...)
			return
		}
		s.Log.Print(append([]interface{}{"Server (Child) : "}, v...)...)
	}
}
func (s *Server) args() []string {
	p := "*.go"
	if s.Dir != "" {
		p = filepath.Join(s.Dir, p)
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
	return args
}

type loggerFunc func(v ...interface{})

func (l loggerFunc) Print(v ...interface{}) {
	l(v...)
}

func (s *Server) Run(main func(c *Runner) error) error {
	if s.id != 0 {
		return fmt.Errorf("Server was already started")
	}
	counter += 1
	s.id = counter

	if s.isParent() {
		s.log("Starting parent")

		s.r.Log = loggerFunc(s.log)
		err := s.startListener()
		if err != nil {
			return err
		}
		if s.AfterListen != nil {
			go s.AfterListen()

		}

		s.r.Command = func() *exec.Cmd {
			cmd := exec.Command("go", s.args()...)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%d", envName, s.id))
			cmd.Stdout = os.Stdout
			cmd.Stdin = os.Stdout
			cmd.Stderr = os.Stdout
			cmd.ExtraFiles = []*os.File{s.fd}
			cmd.Dir = s.Dir
			return cmd
		}
		err = main(&s.r)
		s.r.Stop()
		return err
	}
	s.log("Starting child")
	return s.runChild()
}

func (s *Server) isParent() bool {
	i, err := strconv.Atoi(os.Getenv(envName))
	return err != nil || i != s.id
}

func (s *Server) startListener() error {
	if s.Addr == "" {
		panic("Addr not defined")
	}
	s.log("Start listening in", s.Addr)
	addr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	fd, err := l.File()
	if err != nil {
		l.Close()
		return err
	}

	s.fd = fd
	return nil
}

func (s *Server) runChild() error {
	s.log("Getting the listener")
	l, err := net.FileListener(os.NewFile(3, "listener"))
	if err != nil {
		return err
	}

	s.log("Calling Child")
	return s.Child(l)
}
