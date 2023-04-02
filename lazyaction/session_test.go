package lazyaction

import (
	"net/http"
	"strings"
	"testing"

	"golazy.dev/lazyapp/apptest"
)

func TestSession(t *testing.T) {

	r := &Dispatcher{}

	r.Resource(&SessionController{})

	expect := apptest.New(t, r).Expect

	result := expect("/session/one")
	result.Code(200)

	cookie := result.Headers().Get("Set-Cookie")
	cookieS := strings.Split(cookie, ";")
	if len(cookieS) < 1 {
		t.Fatal("Missing cookie", cookie)
	}
	cookie = cookieS[0]

	expect("/session/two", http.Header{
		"Cookie": {cookie},
	}).Code(200).Body("123")

}
