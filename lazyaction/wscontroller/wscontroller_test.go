package wscontroller

import (
	"testing"

	"golazy.dev/lazyapp"
	"golazy.dev/lazyapp/apptest"
)

type SampleMsg struct {
	Name string
	To   string
}

var disconnects int

type TestController struct {
}

func (tc *TestController) Connect(c *Client) {
	c.Send("welcome")
}
func (tc *TestController) Disconnect(c *Client) {
	disconnects++
}

func (tc *TestController) Hi(c *Client, msg *SampleMsg) {
	c.SendCommand("data", SampleMsg{Name: "me", To: msg.Name})
}

func TestWsController(t *testing.T) {

	a := &lazyapp.App{}
	a.Route("/", NewWSHandler(&TestController{}))

	ws := apptest.New(t, a).Connect("/")

	ws.Expect("welcome")
	ws.SendCommand("Hi", SampleMsg{Name: "world"})

	ws.ExpectCommand("data", SampleMsg{Name: "me", To: "world"})
	ws.Close()

	if disconnects != 1 {
		t.Fatal("disconnects != 1")
	}

}
