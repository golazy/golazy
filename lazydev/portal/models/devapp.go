package models

import (
	"net/url"
	"sync"
)

var (
	appL   sync.Mutex
	appURL *url.URL
)

func AppURL() *url.URL {
	appL.Lock()
	defer appL.Unlock()
	if appURL == nil {
		return nil
	}
	return appURL
}

func AppSetURL(url *url.URL) {
	appL.Lock()
	defer appL.Unlock()
	appURL = url
}
