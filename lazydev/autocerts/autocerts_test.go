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

var (
	// DefaultCS is the default Certificate Server. It holds the ca.pem and
	// ca.key of the certificate authority.  If the files do not exists, the
	// certificate server will generate a
	// selfsigned certificate authority. It is responsablity of the caller to
	// install the authority in the client machine.
	DefaultCS = &CertificateServer{
		CAPemFile: "ca.pem",
		CAKeyFile: "ca.key",
	}
)

// CertificateFromClientHello returns a certifiacte given a tls ClientHelloInfo package
// This ClientHelloInfo is part of the tls handshake and is expose to the http client through
func CertificateFromClientHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return DefaultCS.CertificateFromClientHello(hello)
}

func CertificateFor(domain string) (*tls.Certificate, error) {
	return DefaultCS.CertificateFor(domain)
}

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
	defer func() {
		//os.Remove("test.pem")
		//os.Remove("test.key")
	}()

	server.TLS = &tls.Config{
		GetCertificate: cs.CertificateFromClientHello,
		RootCAs:        certpool,
	}

	server.StartTLS()
	defer func() {
		server.Close()
	}()

	// Configure the client to use the certificate authority
	clientCertPool := x509.NewCertPool()
	clientCertPool.AddCert(cs.CA.cert)

	c := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Second * 60 * 60,
				KeepAlive: time.Second * 60 * 60,
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
