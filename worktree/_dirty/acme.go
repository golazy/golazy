package lazydev

import (
	"crypto/tls"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

type productionServer struct {
	Manager  autocert.Manager
	DirCache autocert.DirCache
}

func (s *Server) serveProduction() error {

	s.DirCache = autocert.DirCache("golazy")
	s.Manager = autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  s.DirCache,
	}

	s.server = &http.Server{
		Addr:    s.HTTPSAddr,
		Handler: s.Handler,
		TLSConfig: &tls.Config{
			GetCertificate: s.Manager.GetCertificate,
		},
	}

	// Start http server to handle acme challenge
	// This will not handle our app
	go http.ListenAndServe(s.HTTPAddr, s.Manager.HTTPHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
		}),
	))
	return s.server.ListenAndServeTLS("", "")
}
