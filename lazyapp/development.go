package lazyapp

import (
	"fmt"
	"net/http"

	"golazy.dev/lazydev"
)

// -build !production,!staging

func (a *App) ListenAndServe() error {
	if a.Server == nil {
		a.Server = &lazydev.Server{
			BootMode: lazydev.ProductionMode,
		}
	}
	fmt.Println(a.Router.String())
	err := a.Server.ListenAndServe()
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8080", a.Router)
}
