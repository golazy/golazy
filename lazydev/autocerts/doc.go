/*
Package autocerts generates certificates for subdomains at runtime

autocerts focus is to generate valid https certificates for development for any domain.

autocerts generates uses a selfsigned certificate by default.

	cs := &autocerts.CertificateServer{
		Subject:   CertSubject // Or leave it nil to use DefaultCeritificateSubject
		CAPemFile: CertPEMPath // Where to load/save the certificate authority
		CAKeyFile: CertKEYPath
	}

	// CertificatePool will load the certificate authority from the pem and key files
	// If it can't be loaded, it will generate one automatically
	certpool, err := cs.CertificatePool()
	if err != nil {
		panic(err)
	}

	// And configure the http server to use it
	srv := &http.Server{
		TLSConfig:    &tls.Config{
			GetCertificate: cs.CertificateFromClientHello,
			RootCAs:        certpool,
		}
		Handler:      myHandler,
	}
*/
package autocerts
