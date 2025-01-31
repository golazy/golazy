# autocerts

## Description

The `autocerts` package generates TLS certificates suitable for the HTTP server with a common certificate authority. It allows developers to create and manage certificates for specific domains at runtime.

## Usage

### Example

```go
package main

import (
	"autocerts"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
)

func main() {
	ac, err := autocerts.LoadOrCreate("my_app_ca.pem", nil)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ac.CACert())

	server := http.Server{
		TLSConfig: &tls.Config{
			GetCertificate: ac.CertificateFromHello,
			RootCAs:        certPool,
		},
		Addr: ":7654",
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})

	go server.ListenAndServeTLS("", "")
	defer server.Shutdown(context.Background())

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
}
```

## Dependencies

- Go 1.16 or later

## Installation

To install the `autocerts` package, run:

```sh
go get github.com/golazy/golazy/autocerts
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub.

## Reporting Issues

If you encounter any issues or have any questions, please open an issue on GitHub.
