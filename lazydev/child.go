package lazydev

import (
	_ "embed"
	"time"

	"crypto/tls"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
)

var DefaultServeMux http.Handler
var DefaultListenAddr = ":3000"

func init() {
	port := os.Getenv("PORT")

	if port != "" {
		if !strings.Contains(port, ":") {
			port = ":" + port
		}
		DefaultListenAddr = port
	}

}

//go:embed cert.pem
var certPem string

//go:embed key.pem
var keyPem string

func childStart() {
	l, err := net.FileListener(os.NewFile(3, "listener"))
	if err != nil {
		log.Fatal(err)
	}

	if !caPresent() {
		err = generateRootCertificate()
		if err != nil {
			log.Fatal(err)
		}
	}

	mux := DefaultServeMux
	if mux == nil {
		mux = http.DefaultServeMux
	}

	handler := handlers.CombinedLoggingHandler(os.Stdout, mux)

	cert, err := tls.X509KeyPair([]byte(certPem), []byte(keyPem))
	if err != nil {
		log.Fatal(err, keyPem)
	}
	certs := make(map[string]*tls.Certificate)
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if cert, ok := certs[hello.ServerName]; ok {
				return cert, nil
			}
			cert, err := getCertificate(hello.ServerName)
			if err != nil {
				return nil, err
			}
			certs[hello.ServerName] = cert
			return cert, nil
		},
	}

	srv := &http.Server{
		TLSConfig:    cfg,
		ReadTimeout:  time.Minute,
		WriteTimeout: time.Minute,
		Handler:      handler,
	}

	err = srv.ServeTLS(l, "", "")
	if err != nil {
		log.Fatal(err)
	}

}
