package models

import (
	"fmt"
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
	fmt.Println("AppSetURL", url)
	fmt.Println("AppSetURL", url)
	fmt.Println("AppSetURL", url)
	fmt.Println("AppSetURL", url)
	fmt.Println("AppSetURL", url)
	fmt.Println("AppSetURL", url)
	appURL = url
}
