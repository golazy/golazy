package autocerts

import (
	"os"
	"testing"
)

func TestCA(t *testing.T) {

	const pemFile = "TestCA.pem"
	const keyFile = "TestCA.key"

	ca, err := GenerateCA(DefaultCertificateSubject, pemFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	err = ca.Save()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Remove(ca.certFile)
		os.Remove(ca.keyFile)
	}()

	ca2, err := LoadCA(pemFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	if !ca2.cert.Equal(ca.cert) {
		t.Error("Certificates aren't equal")
	}

	if !ca2.key.Equal(ca.key) {
		t.Error("Keys aren't equal")
	}
}
