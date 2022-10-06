package autocerts

import (
	"crypto/x509"
	"os"
	"testing"
)

func TestAutocerts(t *testing.T) {
	keyLength = 1024 // speed the tests
	os.Remove("autocerts.pem")
	defer os.Remove("autocerts.pem")

	_, err := Create("autocerts.pem", DefaultSubject)
	if err != nil {
		t.Fatal(err)
	}
	//defer os.Remove("autocerts.pem")

	ac, err := Load("autocerts.pem")
	if err != nil {
		t.Fatal(err)
	}

	cert, err := ac.CertificateFor("example.com")
	if err != nil {
		t.Fatal(err)
	}

	if len(cert.Certificate) != 2 {
		t.Fatal("generated certificate should contain both certificates")
	}

	cert1, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if cert1.Subject.CommonName != "example.com" {
		t.Fatal("Expecting the generated certificate to be for example.com")
	}
}
