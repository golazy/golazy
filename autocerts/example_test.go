package autocerts

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"time"
)

func Example() {

	// import "github.com/golazy/golazy/libs/autocerts"

	// Manually load or create
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
		defer os.Remove("my_app_ca.pem")
	}
	// The LoadOrCreate method could be called to do the same

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
	// Output: hello
}
