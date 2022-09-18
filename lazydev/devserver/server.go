package devserver

import (
	"crypto/tls"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dietsche/rfsnotify"
	"github.com/golazy/golazy/lazydev/devserver/autocerts"
	"github.com/golazy/golazy/lazydev/devserver/protocolmux"
	"github.com/golazy/golazy/lazydev/devserver/tcpdevserver"
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
		watcher, err := rfsnotify.NewWatcher()

		if err != nil {
			log.Fatal(err)
		}

		for _, s := range s.Watch {
			if strings.HasSuffix(s, "/...") {
				err = watcher.AddRecursive(s[:len(s)-4])
			} else {
				err = watcher.Add(s)
			}
			if err != nil {
				log.Fatal("Can't watch " + s + ": " + err.Error())
			}
		}

		// Don't refire constantly
		bufferedMessages := NewDelayer(watcher.Events)
		for {
			select {
			case events, ok := <-bufferedMessages:
				if !ok {
					panic("closed")
				}
				r.Restart()
				fmt.Println(events)
			case err, ok := <-watcher.Errors:
				if !ok {
					panic("closed")
				}
				log.Println("error:", err)
				panic(err)
			}
		}
	})

	return nil
}

func (s *Server) serveHTTP(l net.Listener) {
	http.Serve(l, s.HTTPHandler)
}

func (s *Server) serveHTTPS(l net.Listener) {

	cs := &autocerts.CertificateServer{
		Subject:   s.CertSubject,
		CAPemFile: s.CertPEMPath,
		CAKeyFile: s.CertKEYPath,
	}
	certpool, err := cs.CertificatePool()
	if err != nil {
		panic(err)
	}
	cfg := &tls.Config{
		GetCertificate: cs.CertificateFromClientHello,
		RootCAs:        certpool,
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
