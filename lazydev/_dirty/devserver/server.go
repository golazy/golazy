// package devserver implements and http and https servers with autoreload on files changes and automatic https certificate
package devserver

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"golazy.dev/lazydev/autocerts"
	"golazy.dev/lazydev/filewatcher"
	"golazy.dev/lazydev/protocolmux"
	"golazy.dev/lazydev/tcpdevserver"
)

type Server struct {
	Addr                     string
	Dir                      string
	Watch                    []string
	CertPEMPath, CertKEYPath string
	CertSubject              *pkix.Name
	HTTPHandler              http.Handler
	HTTPSHandler             http.Handler
	AfterRestart             func()
	s                        *tcpdevserver.Server
}

func (s *Server) ListenAndServe() error {
	s.s = &tcpdevserver.Server{
		Addr:  s.Addr,
		Child: s.serve,
		Log:   log.Default(),
	}

	s.s.Run(func(r *tcpdevserver.Runner) error {
		r.Start()
		r.AfterRestart = s.AfterRestart

		fw, err := filewatcher.New(s.Dir)
		if err != nil {
			panic(err)
		}
		changes, err := fw.Watch()
		if err != nil {
			panic(err)
		}
		for range changes {
			r.Restart()
		}
		return nil
	})

	return nil
}

func (s *Server) serveHTTP(l net.Listener) {
	http.Serve(l, s.HTTPHandler)
}

func (s *Server) serveHTTPS(l net.Listener) {

	ac, err := autocerts.Load(s.CertPEMPath)
	if err != nil {
		// Fail if the error is diferent that file not found
		var pathError *fs.PathError
		if !errors.As(err, &pathError) {
			panic(err)
		}

		// Create the certificate
		ac, err = autocerts.Create(s.CertPEMPath, nil)
		if err != nil {
			panic(err)
		}
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	cfg := &tls.Config{
		GetCertificate: ac.CertificateFromHello,
		RootCAs:        certPool,
	}

	srv := &http.Server{
		TLSConfig:    cfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		Handler:      s.HTTPSHandler,
	}

	srv.ServeTLS(l, "", "")
}

func (s *Server) serve(l net.Listener) error {
	m := &protocolmux.Mux{L: l}
	s.s.Log.Print("Registering http")

	go s.serveHTTP(m.ListenTo(protocolmux.HTTPPrefix))
	go s.serveHTTPS(m.ListenTo(protocolmux.TLSPrefix))

	return m.Listen()
}
