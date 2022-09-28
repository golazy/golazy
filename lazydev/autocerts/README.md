# autocerts

Package autocerts generates tls certificate suitable for the http server with a common certificate authority

## Variables

DefaultCertificateSubject is used if no subject is supplied

```golang
var DefaultCertificateSubject = &pkix.Name{
    Organization:  []string{"autocerts"},
    Country:       []string{"DE"},
    Province:      []string{"Berlin"},
    Locality:      []string{"Berlin"},
    StreetAddress: []string{""},
    PostalCode:    []string{""},
}
```

## Types

### type [Autocerts](/autocerts.go#L33)

`type Autocerts struct { ... }`

Autocerts generates certificates for specific domains at runtime.
It can be use through a provided Certificate Authority or through a generated one.

#### func [Create](/autocerts.go#L48)

`func Create(caFile string, subject *pkix.Name) (*Autocerts, error)`

Create creates a new CA with the given subject. If subject is nil, DefaultCertificateSubejct will be used.
Once the certificate is create, it is saved in caFile

#### func [Load](/autocerts.go#L68)

`func Load(caFile string) (*Autocerts, error)`

Load reads the caFile.
The file should be in pem format and should contain a certificate and a rsa private key
If the file can't be found, the error will contain fs.PathError

#### func (*Autocerts) [CACert](/autocerts.go#L42)

`func (ac *Autocerts) CACert() *x509.Certificate`

CACert returns the Certificate Authority

#### func (*Autocerts) [CertificateFor](/autocerts.go#L196)

`func (ac *Autocerts) CertificateFor(domain string) (*tls.Certificate, error)`

CertificateFor returns a valid tls certificate for the given domain

#### func (*Autocerts) [CertificateFromHello](/autocerts.go#L191)

`func (ac *Autocerts) CertificateFromHello(hello *tls.ClientHelloInfo) (*tls.Certificate, error)`

CertificateFromHello returns a valid tls certificate for the given server name inside the tls clientHelloInfo
It is meant to be used inside tls.TLSConfig as GetCertificate

## Examples

```golang
// import "github.com/golazy/golazy/lazydev/autocerts"

// Configure Autocerts
ac, err := Load("my_app_ca.pem")

if err != nil {
    // Fail if the error is diferent that file not found
    var pathError *fs.PathError
    if !errors.As(err, &pathError) {
        panic(err)
    }

    // Create the certificate
    ac, err = Create("my_app_ca.pem", nil)
    if err != nil {
        panic(err)
    }
}

// Configure http server
certPool := x509.NewCertPool()
certPool.AddCert(ac.CACert())

server := http.Server{
    TLSConfig: &tls.Config{
        GetCertificate: ac.CertificateFromHello,
        RootCAs:        certPool,
    },
    Addr: ":7654",
}

// Add a handler
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello"))
})

// And start the server
go server.ListenAndServeTLS("", "")
defer server.Shutdown(context.Background())

// Now lets try to do a request

// Wait for the serer to start
time.Sleep(time.Second)

client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: &tls.Config{
            RootCAs: certPool,
        },
    },
}

res, err := client.Get("https://localhost:7654/")
if err != nil {
    panic(err)
}
defer res.Body.Close()

body, err := io.ReadAll(res.Body)
if err != nil {
    panic(err)
}

fmt.Println(string(body))
```

 Output:

```
hello
```

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
