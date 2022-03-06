package autocerts

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAutocerts(t *testing.T) {

	tlsCert, err := DefaultCS.CertificateFor("localhost")
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}

	if cert.DNSNames[0] != "localhost" {
		t.Fatal(err)
	}

}

func TestServer(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})
	server := httptest.NewUnstartedServer(handler)

	cs := &CertificateServer{
		Subject:   nil,
		CAPemFile: "test.pem",
		CAKeyFile: "test.key",
	}
	certpool, err := cs.CertificatePool()
	if err != nil {
		t.Error(cs.CA.cert.Raw)
		t.Fatal(err)
	}

	server.TLS = &tls.Config{
		GetCertificate: cs.CertificateFromClientHello,
		RootCAs:        certpool,
	}

	server.StartTLS()
	defer func() {
		server.Close()
	}()

	clientCertPool := x509.NewCertPool()
	clientCertPool.AddCert(cs.CA.cert)

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Second,
				KeepAlive: time.Second,
			}).DialContext,
			TLSClientConfig: &tls.Config{
				RootCAs: clientCertPool,
			},
		},
	}

	_, err = c.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}

}
