package lazydev

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/adrg/xdg"
)

// https://shaneutt.com/blog/golang-ca-and-signed-cert-go/

const (
	caPEMName = "ca.pem"
	caKEYName = "ca.key"
)

func caPresent() bool {
	_, err := readFile(caPEMName)
	if err != nil {
		return false
	}
	_, err = readFile(caKEYName)
	return err == nil
}

func getCertificate(domain string) (*tls.Certificate, error) {
	//https://godocs.io/crypto/x509#Certificate
	// Create Certificate
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"golazy"},
			Country:       []string{"DE"},
			Province:      []string{"Berlin"},
			Locality:      []string{"Berlin"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		DNSNames: []string{domain},
		//IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Create key
	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}
	// Get ca
	data, err := readFile(caPEMName)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.New("Can't parse ca certificate")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("Ca certificate of unknown type %v", block.Type)
	}

	ca, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	// Get caPrivKey
	data, err = readFile(caKEYName)
	if err != nil {
		return nil, err
	}
	block, _ = pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.New("Can't parse ca key")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("Ca certificate of unknown type %v", block.Type)
	}

	caPrivKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// SignCertificate
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// Get new Cert PEM
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Get new Cert Key
	certPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	// Create certificate
	domainCert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err == nil {
		log.Println("Succesfully generate certificate for", domain)
	}
	return &domainCert, err
}

func generateRootCertificate() error {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(20220224),
		Subject: pkix.Name{
			Organization:  []string{"GoLazy"},
			Country:       []string{"DE"},
			Province:      []string{"Berlin"},
			Locality:      []string{"Berlin"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	// Certificate
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})
	caPEMPath, err := saveFile(caPEMName, caPEM.String())
	if err != nil {
		return err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})
	caPrivKeyPEMPath, err := saveFile(caKEYName, caPrivKeyPEM.String())
	if err != nil {
		return err
	}

	log.Printf("Generated CA certificate:")
	log.Printf("  PEM: %s", caPEMPath)
	log.Printf("  KEY: %s", caPrivKeyPEMPath)
	return nil
}

func readFile(file string) (string, error) {
	errfn := func(err error) error { return fmt.Errorf("can't read file %s: %w", file, err) }
	filepath, err := xdg.DataFile("golazy/" + file)
	if err != nil {
		return "", errfn(err)
	}
	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", errfn(err)
	}
	return string(data), nil
}

func saveFile(file, content string) (string, error) {
	errfn := func(err error) error { return fmt.Errorf("can't save file %s: %w", file, err) }
	filepath, err := xdg.DataFile("golazy/" + file)
	if err != nil {
		return filepath, errfn(err)
	}

	f, err := os.Create(filepath)
	if err != nil {
		return filepath, errfn(err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return filepath, errfn(err)
	}
	return filepath, nil
}
