package multihttp

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"

	"golazy.dev/protocolmux"
)

type tcp struct {
	Addr      string
	TLSConfig *tls.Config
	Handler   http.Handler
	pm        *protocolmux.Mux
	http      *http.Server
	https     *http.Server
	l         *net.TCPListener
}

func (s *tcp) ListenAndServe() error {
	// listen
	if s.Addr == "" {
		return errors.New("no address specified")
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", s.Addr)
	if err != nil {
		return err
	}
	s.l, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	s.pm = &protocolmux.Mux{L: s.l}
	httpListener := s.pm.ListenTo(protocolmux.HTTPPrefix)
	httpsListener := s.pm.ListenTo(protocolmux.TLSPrefix)

	// HTTP server
	s.http = &http.Server{
		Handler: s.Handler,
	}

	// HTTPS server
	s.https = &http.Server{
		Handler:   s.Handler,
		TLSConfig: s.TLSConfig,
	}

	errs := collectErrors(
		s.pm.Listen,
		func() error { return s.https.ServeTLS(httpsListener, "", "") },
		func() error { return s.http.Serve(httpListener) },
	)

	err = filterError(errs, http.ErrServerClosed)
	if err == nil {
		return http.ErrServerClosed
	}
	return errs
}

func (s *tcp) Close() error {

	a := func(msg string, fn func() error) func() error {

		return func() error {
			err := fn()
			//fmt.Println(msg, err)
			return err
		}
	}

	var errs []error
	err := collectErrorsSync(
		a("http", s.http.Close),
		a("https", s.https.Close),
		a("pm", s.pm.Close),
	)
	eachError(err, func(err error) {
		if !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	})

	return errors.Join(errs...)
}
