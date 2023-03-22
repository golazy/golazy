// Package browsercommand enable browser to call go code
package browsercommand

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

type BrowserCommand struct {
	// Endpoint is the registered route
	Endpoint string
	Api      any
}

var DefaultEndpoint = "/_golazy/commands"

// New will create a BrowserCommand
func New(api any) *BrowserCommand {
	bc := &BrowserCommand{
		Endpoint: DefaultEndpoint,
		Api:      api,
	}
	bc.RegisterHandlers()
}

// RegisterHandlers register BrowserCommand
func (bc *BrowserCommand) RegisterHandlers() {

}

func (bc *BrowserCommand) ServeHTTP(w http.ResponseWriter, r *http.Request) {

}

func (bc *BrowserCommand) serve()

type TestApi struct{}

func (api *TestApi) Hello(name string, age int) (error, string) {
	return nil, fmt.Sprintf("Hello %n\nHappy that you are %d", name, age)
}

func TestBrowserCommand(t *testing.T) {

	New(&TestApi)
}
