package lazydev

import (
	"crypto/x509/pkix"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/adrg/xdg"
	"github.com/guillermo/golazy/lazydev/devserver"
)

func certPEM() string {
	certPEM, err := xdg.DataFile("golazy/ca.pem")
	if err != nil {
		return "ca.pem"
	}
	return certPEM
}

func Serve(h http.Handler) {
	addr := ":3000"
	if port := os.Getenv("PORT"); port != "" {
		if strings.ContainsRune(port, ':') {
			addr = port
		} else {
			addr = ":" + port
		}
	}

	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	certKEY, err := xdg.DataFile("golazy/ca.key")
	if err != nil {
		certKEY = "ca.key"
	}

	watchList := []string{wd}
	paths := os.Getenv("LAZYWATCH")
	if paths != "" {
		watchList = strings.Split(paths, ",")
	}

	s := &devserver.Server{
		Addr:        addr,
		Dir:         wd,
		Watch:       watchList,
		CertPEMPath: certPEM(),
		CertKEYPath: certKEY,
		CertSubject: &pkix.Name{
			Organization:       []string{"golazy"},
			OrganizationalUnit: []string{"lazydev"},
			Country:            []string{"DE"},
			Province:           []string{"Berlin"},
			Locality:           []string{"Berlin"},
			StreetAddress:      []string{""},
			PostalCode:         []string{""},
		},
		HTTPHandler:  httpHandler(certPEM(), certKEY),
		HTTPSHandler: httpsHandler(h),
		AfterRestart: func() {
			fmt.Println("Serve.Affter Restart called")

			control.Broadcast(msg{Command: "reload"})
		},
	}

	err = s.ListenAndServe()
	if err != nil {
		panic(err)
	}

}
