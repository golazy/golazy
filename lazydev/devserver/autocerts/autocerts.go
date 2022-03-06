package autocerts

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"sync"
	"time"
)

var (
	DefaultCS = &CertificateServer{
		CAPemFile: "ca.pem",
		CAKeyFile: "ca.key",
	}
)

func CertificateFromClientHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return DefaultCS.CertificateFromClientHello(hello)
}

func CertificateFor(domain string) (*tls.Certificate, error) {
	return DefaultCS.CertificateFor(domain)
}

type CertificateServer struct {
	sync.Mutex
	Subject   *pkix.Name
	CAPemFile string
	CAKeyFile string
	CA        *CAFiles
	cbh       map[string]*tls.Certificate // Certificates by host
}

func (c *CertificateServer) CertificateFromClientHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return c.CertificateFor(hello.ServerName)
}

func (c *CertificateServer) CertificatePool() (*x509.CertPool, error) {
	certPool := x509.NewCertPool()
	if c.CA == nil {
		ca, err := LoadOrGenerateCA(c.Subject, c.CAPemFile, c.CAKeyFile)
		if err != nil {
			return nil, err
		}
		c.CA = ca
	}

	certPool.AddCert(c.CA.cert)

	return certPool, nil
}

func (c *CertificateServer) CertificateFor(domain string) (*tls.Certificate, error) {
	c.Lock()
	defer c.Unlock()
	if c.cbh == nil {
		c.cbh = make(map[string]*tls.Certificate)
	}

	if cert, ok := c.cbh[domain]; ok {
		return cert, nil
	}

	cert, err := c.generateCertificateFor(domain)
	if err != nil {
		return cert, err
	}
	c.cbh[domain] = cert
	return cert, nil

}

func (c *CertificateServer) generateCertificateFor(domain string) (*tls.Certificate, error) {
	var subject pkix.Name
	if c.Subject != nil {
		subject = *c.Subject
	} else {
		subject = *DefaultCertificateSubject
	}

	subject.OrganizationalUnit = []string{domain}
	subject.CommonName = domain

	xCert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject:      subject,
		DNSNames:     []string{domain},
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

	if c.CA == nil {
		ca, err := LoadOrGenerateCA(c.Subject, c.CAPemFile, c.CAKeyFile)
		if err != nil {
			return nil, err
		}
		c.CA = ca
	}

	// SignCertificate
	certBytes, err := x509.CreateCertificate(rand.Reader, xCert, c.CA.cert, &certPrivKey.PublicKey, c.CA.key)
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
	cert, err := tls.X509KeyPair(certPEM.Bytes(), certPrivKeyPEM.Bytes())
	if err == nil {
		log.Println("Succesfully generate certificate for", domain)
	}
	return &cert, err
}
