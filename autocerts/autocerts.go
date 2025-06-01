// Package autocerts generates tls certificate suitable for the http server with a common certificate authority
package autocerts

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
	"io/fs"
	"math/big"
	"os"
	"sync"
	"time"
)

var keyLength = 1024

// DefaultSubject is used if no subject is supplied
var DefaultSubject = &pkix.Name{
	Organization:  []string{"golazy"},
	Country:       []string{"DE"},
	Province:      []string{"Berlin"},
	Locality:      []string{"Berlin"},
	StreetAddress: []string{""},
	PostalCode:    []string{""},
}

// Autocerts generates certificates for specific domains at runtime.
// It can be use through a provided Certificate Authority or through a generated one.
type Autocerts struct {
	caFile string
	caCert *x509.Certificate
	caKey  *rsa.PrivateKey
	sync.Mutex
	certCache map[string]*tls.Certificate
}

// CACert returns the Certificate Authority
func (ac *Autocerts) CACert() *x509.Certificate {
	return ac.caCert
}

// Create creates a new CA with the given subject. If subject is nil, DefaultCertificateSubejct will be used.
// Once the certificate is create, it is saved in caFile
func Create(caFile string, subject *pkix.Name) (*Autocerts, error) {

	ac := &Autocerts{
		caFile: caFile,
	}

	if err := ac.generateCA(subject); err != nil {
		return nil, err
	}
	if err := ac.saveCA(); err != nil {
		return nil, err
	}

	return ac, nil

}

// Load reads the caFile.
// The file should be in pem format and should contain a certificate and a rsa private key
// If the file can't be found, the error will contain fs.PathError
func Load(caFile string) (*Autocerts, error) {

	ac := &Autocerts{
		caFile: caFile,
	}

	data, err := os.ReadFile(ac.caFile)
	if err != nil {
		return nil, fmt.Errorf("can't read CA file %s: %w", ac.caFile, err)
	}

	ac.caCert, ac.caKey, err = DecodeCAPem(data)
	if err != nil {
		return nil, err
	}

	return ac, nil
}

// LoadOrCreate tries to Load the certificate. If it does not exists, it will create one.
// If subject is nil, it will use DefaultSubject
func LoadOrCreate(certPath string, subject *pkix.Name) (*Autocerts, error) {
	ac, err := Load(certPath)
	if err == nil {
		return ac, err
	}
	var pathError *fs.PathError
	if !errors.As(err, &pathError) {
		return nil, err
	}
	return Create(certPath, subject)
}

func eachBlock(data []byte) (blocks []*pem.Block) {
	var p *pem.Block
	for {
		p, data = pem.Decode(data)
		if p == nil {
			return blocks
		}
		blocks = append(blocks, p)
	}
}

func DecodeCAPem(data []byte) (cert *x509.Certificate, key *rsa.PrivateKey, errs error) {
	var err error
	for _, b := range eachBlock(data) {
		switch b.Type {
		case "CERTIFICATE":
			cert, err = x509.ParseCertificate(b.Bytes)
			if err != nil {
				errs = errors.Join(errs, err)
			}
		case "RSA PRIVATE KEY":
			key, err = x509.ParsePKCS1PrivateKey(b.Bytes)
			if err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return
}

func (ac *Autocerts) saveCA() error {
	// Encode certificate
	caPEM := new(bytes.Buffer)
	pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ac.caCert.Raw,
	})

	pem.Encode(caPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(ac.caKey),
	})

	// Create certificate file
	f, err := os.OpenFile(ac.caFile, os.O_WRONLY|os.O_EXCL|os.O_CREATE, 0700)
	if err != nil {
		return fmt.Errorf("can't create CA in %s: %w", ac.caFile, err)
	}
	defer f.Close()

	// Save certificate
	_, err = f.WriteString(caPEM.String())
	if err != nil {
		return fmt.Errorf("can't write CA file in %s:%q", ac.caFile, err)
	}

	return nil
}

func (ac *Autocerts) generateCA(subject *pkix.Name) error {
	if subject == nil {
		subject = DefaultSubject
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
	privateKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return fmt.Errorf("can't generate rsa key pair: %q", err)
	}

	// Get certificate bytes
	caBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("can't create certificate: %q", err)
	}

	cert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return fmt.Errorf("can't parse created certificate: %q", err)
	}

	ac.caCert = cert
	ac.caKey = privateKey

	return nil
}

// CertificateFromHello returns a valid tls certificate for the given server name inside the tls clientHelloInfo
// It is meant to be used inside tls.TLSConfig as GetCertificate
func (ac *Autocerts) CertificateFromHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return ac.CertificateFor(hello.ServerName)
}

// CertificateFor returns a valid tls certificate for the given domain
func (ac *Autocerts) CertificateFor(domain string) (*tls.Certificate, error) {
	ac.Lock()
	defer ac.Unlock()

	if ac.certCache == nil {
		ac.certCache = make(map[string]*tls.Certificate)
	} else {
		cert, ok := ac.certCache[domain]
		if ok {
			return cert, nil
		}
	}

	cert, err := ac.generateCertFor(domain)
	if err != nil {
		return cert, err
	}
	ac.certCache[domain] = cert
	return cert, nil
}

func (ac *Autocerts) generateCertFor(domain string) (*tls.Certificate, error) {
	subject := ac.caCert.Subject

	subject.OrganizationalUnit = []string{domain}
	subject.CommonName = domain

	xCert := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Issuer:       ac.caCert.Subject,
		Subject:      subject,
		DNSNames:     []string{domain},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// Create key
	certPrivKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, err
	}

	// SignCertificate
	certBytes, err := x509.CreateCertificate(rand.Reader, xCert, ac.caCert, &certPrivKey.PublicKey, ac.caKey)
	if err != nil {
		return nil, err
	}

	// Encode
	certPEM := new(bytes.Buffer)
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Append ca certificate
	pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ac.caCert.Raw,
	})

	// Get new Cert Key
	certKeyPEM := new(bytes.Buffer)
	pem.Encode(certKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	})

	// Create certificate
	cert, err := tls.X509KeyPair(certPEM.Bytes(), certKeyPEM.Bytes())
	return &cert, err

}

// TLSConfigFile is a helper that returns a tls config with the certificate from the file
// If the file does not exists, a new certificate is created
func TLSConfigFile(path string) (*tls.Config, error) {

	ac, err := LoadOrCreate(path, nil)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	return &tls.Config{
		GetCertificate: ac.CertificateFromHello,
		RootCAs:        certPool,
	}, nil

}
