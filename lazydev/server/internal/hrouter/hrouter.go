// Package hrouter provides a http/https handler router for lazydev server.
// It allows to specify the follwoing handlers:
// - HTTPHandler: the handler for HTTP requests
// - PrefixHandler: the handler for https requests with a given prefix
// - FallbackHandler: the handler for https request when the server is not running
package hrouter

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/adrg/xdg"
	"golazy.dev/lazydev/autocerts"
	"golazy.dev/lazydev/protocolmux"
	"golazy.dev/lazydev/server/internal/cbhandler"
)

type Server struct {
	cb              *cbhandler.Handler
	HTTPHandler     http.Handler
	PrefixHandler   http.Handler
	Prefix          string
	FallbackHandler http.Handler
	Addr            string
	pm              *protocolmux.Mux
	l               net.Listener
	httpServer      *http.Server
	httpsServer     *http.Server
}

func (s *Server) CBOpen(url *url.URL) {
	s.cb.Open(url)
}
func (s *Server) CBClose() {
	if s.cb != nil {
		s.cb.Close()
	}
}

func (s *Server) Close() error {
	s.httpServer.Close()
	s.httpsServer.Close()
	s.pm.Close()
	return nil
}

// ListenAndServe starts the HTTP and HTTPS servers.
func (s *Server) ListenAndServe() error {
	// listen
	if s.Addr == "" {
		s.Addr = ":2000"
	} else if !strings.Contains(s.Addr, ":") {
		s.Addr = ":" + s.Addr
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.l, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	// Setup fallback handler
	s.cb = cbhandler.New(s.FallbackHandler)

	// Setup muxer
	s.pm = &protocolmux.Mux{L: s.l}
	httpListener := s.pm.ListenTo(protocolmux.HTTPPrefix)
	httpsListener := s.pm.ListenTo(protocolmux.TLSPrefix)

	// HTTP server
	s.httpServer = &http.Server{
		Handler: s.HTTPHandler,
	}
	go s.httpServer.Serve(httpListener)

	// HTTPS server
	cert, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		cert = "golazy.pem"
	}
	log.Println("Certificate Authority stored in:", cert)

	ac, err := autocerts.LoadOrCreate(cert, nil)
	if err != nil {
		return err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	mux := http.NewServeMux()
	mux.Handle(s.Prefix, s.PrefixHandler)
	mux.Handle("/", s.cb)

	s.httpsServer = &http.Server{
		Handler: mux,
		TLSConfig: &tls.Config{
			GetCertificate: ac.CertificateFromHello,
			RootCAs:        certPool,
		},
	}

	go s.httpsServer.ServeTLS(httpsListener, "", "")

	return s.pm.Listen()
}
