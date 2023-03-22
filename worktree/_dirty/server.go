package lazydev

import (
	"context"
	"log"
	"net/http"
)

type bootMode int

const (
	ParentMode bootMode = iota
	ChildMode
	ProductionMode
)

type Server struct {
	BootMode            bootMode
	HTTPAddr, HTTPSAddr string
	Handler             http.Handler
	productionServer
	server *http.Server
	ClientCmd string
	ClientWD string
}

var DefaultServer = &Server{
	BootMode:  ParentMode,
	HTTPAddr:  ":3000",
	HTTPSAddr: ":3000",
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) IsProduction() bool {
	return s.BootMode == ProductionMode
}

func (s *Server) ListenAndServe() error {
	switch s.BootMode {
	case ProductionMode: // Listen on 443 and 80 with lego(acme)
		log.Println("Starting lazydev in production mode")
		return s.serveProduction()
	case ParentMode: // Listen on tcp 3000, builds, start the child and pass the fd
		log.Println("Starting lazydev in parent mode")
		return s.startParent()
	case ChildMode: // Takes the fd 3 and listen on http and https
		log.Println("Starting lazydev in child mode")
		return s.serveChild()
	default:
		panic("Unknown boot mode")
	}
}

func IsProduction() bool {
	return DefaultServer.IsProduction()
}
