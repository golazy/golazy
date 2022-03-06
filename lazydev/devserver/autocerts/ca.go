package autocerts

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"
)

var DefaultCertificateSubject = &pkix.Name{
	Organization:  []string{"autocerts"},
	Country:       []string{"DE"},
	Province:      []string{"Berlin"},
	Locality:      []string{"Berlin"},
	StreetAddress: []string{""},
	PostalCode:    []string{""},
}

type CAFiles struct {
	// TODO: Make it a normal tls.Certificate
	certFile, keyFile string

	cert *x509.Certificate
	key  *rsa.PrivateKey
}

func LoadOrGenerateCA(subject *pkix.Name, certFile, keyFile string) (*CAFiles, error) {
	ca, err := LoadCA(certFile, keyFile)
	if err == nil {
		return ca, nil
	}
	ca, err = GenerateCA(subject, certFile, keyFile)
	if err != nil {
		return ca, err
	}
	err = ca.Save()
	return ca, err
}

func LoadCA(certFile, keyFile string) (*CAFiles, error) {
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("certFile and keyFile must be set")
	}

	// Cert
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("Can't read cerfile %s: %q", certFile, err)
	}
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("ca file %s is empty", certFile)
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("Ca certificate %s of unknown type. %v", certFile, block.Type)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Can't parse certificate in %s: %q", certFile, err)
	}
	// Key
	data, err = os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("Can't read key %s: %q", keyFile, err)
	}

	block, _ = pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("key file %s is empty", keyFile)
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("key file %s have a type of %s. \"RSA PRIVATE KEY\" expected", keyFile, block.Type)
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("Can't parse key in %s: %q", keyFile, err)
	}

	return &CAFiles{
		certFile: certFile,
		keyFile:  keyFile,
		cert:     cert,
		key:      key,
	}, nil

}

func (c *CAFiles) Save() error {
	// Encode certificate
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: c.cert.Raw,
	})

	// Create certificate file
	f, err := os.Create(c.certFile)
	if err != nil {
		return fmt.Errorf("Can't create CA file %q in %q", c.certFile, err)
	}
	defer f.Close()

	// Save certificate
	_, err = f.WriteString(caPEM.String())
	if err != nil {
		return fmt.Errorf("Can't write Certificate Authority file in %s:%q", c.certFile, err)
	}

	// Encode key
	caPrivKeyPEM := new(bytes.Buffer)
	pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(c.key),
	})

	// Create key file
	f, err = os.Create(c.keyFile)
	if err != nil {
		return fmt.Errorf("Can't create CA KEY file in %s:%q", c.keyFile, err)
	}
	defer f.Close()

	_, err = f.WriteString(caPrivKeyPEM.String())
	if err != nil {
		return fmt.Errorf("Can't write CA KEY file in %s:%q", c.keyFile, err)
	}
	return nil
}

func GenerateCA(subject *pkix.Name, certFile, keyFile string) (*CAFiles, error) {
	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("certFile and keyFile must be set")
	}

	if subject == nil {
		subject = DefaultCertificateSubject
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().Unix()),
		Subject:               *subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// Create key
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("can't generate rsa key pair: %q", err)
	}

	// Get certificate bytes
	caBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("can't create certificate: %q", err)
	}

	cert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, fmt.Errorf("can't parse created certificate: %q", err)

	}

	return &CAFiles{
		certFile: certFile,
		keyFile:  keyFile,
		cert:     cert,
		key:      key,
	}, nil
}
