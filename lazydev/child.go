package lazydev

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/adrg/xdg"
	"github.com/golazy/golazy/lazydev/autocerts"
	"github.com/golazy/golazy/lazydev/protocolmux"
)

type child struct {
}

func (s *child) Serve(h http.Handler) error {

	listenerFile := os.NewFile(3, "listener")
	if listenerFile == nil {
		return fmt.Errorf("Expecting listener in FD 3")
	}

	l, err := net.FileListener(listenerFile)
	if err != nil {
		return err
	}
	pm := protocolmux.Mux{L: l}

	// Setup http server
	httpServer := http.Server{
		Handler: h,
	}

	go func() {
		err := httpServer.Serve(pm.ListenTo(protocolmux.HTTPPrefix))
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	// Setup https server
	cert, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		cert = "golazy.pem"
	}
	log.Println("Certificate Authority stored in:", cert)

	ac, err := autocerts.LoadOrCreate(cert, nil)
	if err != nil {
		return err
	}

	log.Println("ac", ac)

	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	httpsServer := http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: ac.CertificateFromHello,
			RootCAs:        certPool,
		},
	}

	log.Println("https server", httpsServer)

	go func() {
		log.Println("Starting https server")
		err := httpsServer.ServeTLS(pm.ListenTo(protocolmux.TLSPrefix), "", "")
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
		log.Println("done with https")
	}()

	err = pm.Listen()
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	log.Println("Ready")
	<-c
	log.Println("Got interrupt signal")

	return nil
}
