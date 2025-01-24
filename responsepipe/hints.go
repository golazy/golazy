//go:build exclude

package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"time"

	"golazy.dev/autocerts"
)

func main() {

	ac, err := autocerts.LoadOrCreate("tmp/test_cert.pem", nil)
	if err != nil {
		panic(err)
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	server := http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: ac.CertificateFromHello,
			RootCAs:        certPool,
		},
		Addr: ":3000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Link", "</css/app.css>; rel=preload; as=style;")
			w.Header().Add("Link", "</js/app.js>; rel=preload; as=script;")
			w.Header().Add("Trailer", "ETag")
			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(http.StatusEarlyHints)
			time.Sleep(time.Second * 5)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello"))
			w.(http.Flusher).Flush()

			time.Sleep(time.Second * 5)
			w.Write([]byte("world!"))
			w.(http.Flusher).Flush()

			time.Sleep(time.Second * 5)
			w.Header().Set("ETag", "123")
		}),
	}

	log.Fatal(server.ListenAndServeTLS("", ""))

}
