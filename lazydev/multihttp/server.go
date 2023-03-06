package multihttp

import (
	"crypto/tls"
	"errors"
	"net/http"
)

type Server struct {
	Addr      string
	TLSConfig *tls.Config
	Handler   http.Handler
	tcp
	udp
}

func (s *Server) ListenAndServe() error {

	s.tcp.Addr = s.Addr
	s.tcp.TLSConfig = s.TLSConfig
	s.tcp.Handler = s.Handler

	s.udp.Addr = s.Addr
	s.udp.TLSConfig = s.TLSConfig
	s.udp.Handler = s.Handler

	err := collectErrors(
		s.tcp.ListenAndServe,
		s.udp.ListenAndServe,
	)

	errs := []error{}
	eachError(err, func(err error) {
		if !errors.Is(http.ErrServerClosed, err) &&
			err.Error() != "quic: Server closed" {
			errs = append(errs, err)
		}
	})
	if len(errs) == 0 {
		return http.ErrServerClosed
	}

	return errors.Join(errs...)
}

func (s *Server) Close() error {
	return collectErrors(
		s.tcp.Close,
		s.udp.Close,
	)
}
