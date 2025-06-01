package multihttp

import (
	"crypto/tls"
	"net/http"

	"github.com/quic-go/quic-go/http3"
)

type udp struct {
	Addr      string
	TLSConfig *tls.Config
	Handler   http.Handler
	s         http3.Server
}

func (s *udp) ListenAndServe() error {
	s.s = http3.Server{
		Handler:   s.Handler,
		Addr:      s.Addr,
		TLSConfig: s.TLSConfig,
	}

	return s.s.ListenAndServe()
}

func (s *udp) Close() error {
	return s.s.Close()
}
