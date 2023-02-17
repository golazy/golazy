package devserver

import (
	"net"
	"net/http"
	"os"
	"strings"
)

type App struct {
	PWD           string
	Path          string
	Tags          []string
	CFlags        []string
	LDFlags       []string
	CompileOutput string
}

func (a *App) Compile() error {
	return nil
}

func (a *App) Start(l net.Listener) error {
	return nil
}

func (a *App) Stop() error {
	return nil
}

func (a *App) Restart() error {
}

type DevServer struct {
	App     *App
	Handler http.Handler
	Addr    string
	err     error
	l       net.Listener
}

func NewDevServer() *DevServer {
	return &DevServer{}
}

type nextState func() nextState

func (s *DevServer) listenTCP() nextState {
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = s.Addr
	} else {
		if !strings.Contains(addr, ":") {
			addr = ":" + addr
		}
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		s.err = err
		return nil
	}
	l, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		s.err = err
		return nil
	}
	s.l = l

	return s.serveHTTP
}

func (s *DevServer) serveHTTP() nextState {
	// Start listening for requests

	return s.compileAndBoot
}

func (s *DevServer) compileAndBoot() nextState {
	err := s.App.Compile()
	if err != nil {
		//notify compilation failed
		// wait for file changes or manual signal
		return s.compileAndBoot
	}
	err = s.App.Start()
	if err != nil {
		// notify that app can't be started
		//wait for file changes or manual signal
	}

	// Stop listening in the parent

	return s.running
}

func (s *DevServer) running() nextState {
	// wait for file changes
	return nil
}

func (s *DevServer) compilationFailed() error {
	return nil
}

func (s *DevServer) ListenAndServe() error {

	state := s.listenTCP()

	for state != nil {
		state = state()
	}

}
