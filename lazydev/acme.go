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

func (s *server) serveProduction(h http.Handler) error {

	s.DirCache = autocert.DirCache("golazy")
	s.Manager = autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  s.DirCache,
	}

	server := &http.Server{
		Addr:    s.HTTPSAddr,
		Handler: h,
		TLSConfig: &tls.Config{
			GetCertificate: s.Manager.GetCertificate,
		},
	}

	go http.ListenAndServe(s.HTTPAddr, s.Manager.HTTPHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
		}),
	))
	return server.ListenAndServeTLS("", "")
}
