package portal

import (
	"crypto/tls"
	"portal/models"

	"github.com/adrg/xdg"
	"golazy.dev/lazyapp"
	"golazy.dev/lazydev/autocerts"
	"golazy.dev/lazydev/devserver/events"
	"golazy.dev/lazydev/multihttp"
)

type PortalApp struct {
	lazyapp.App
	s *multihttp.Server
}

func (a *PortalApp) Event(e events.Event) {
	models.EventSave(e)
}

func (a *PortalApp) ListenAndServe(addr string) error {

	a.s = &multihttp.Server{
		Addr:      addr,
		TLSConfig: getTLSConfig(),
		Handler:   a,
	}
	a.App.Init()
	return a.s.ListenAndServe()
}

func (a *PortalApp) Close() error {
	return a.s.Close()
}

func getTLSConfig() *tls.Config {
	file, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		file = "golazy.pem"
	}

	tls, err := autocerts.TLSConfigFile(file)
	if err != nil {
		panic(err)
	}
	return tls
}
