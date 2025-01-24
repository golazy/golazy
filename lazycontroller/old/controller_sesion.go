package lazycontroller

import (
	"fmt"
	"net/http"

	"golazy.dev/lazysupport"

	"github.com/gorilla/sessions"
)

var Store = sessions.NewCookieStore([]byte("Tjz@-QNkekP2KH8oF9A_GNssbwvftXqv"))
var SessionName = "lazy_session"

func (c *Controller) SessionGet(key string) *lazysupport.Value {
	value, ok := c.session.Values[key]
	if !ok {
		return lazysupport.NewValue(nil)
	}

	return lazysupport.NewValue(value)
}

// SessionValues return all values in the session.
// Modifying the returned map may or may not work.
// Use SessionSet or SessionDelete instead.
func (c *Controller) SessionValues() map[any]any {
	return c.session.Values
}

func (c *Controller) SessionDelete(key string) {
	c.sessionHasChanges = true
	delete(c.session.Values, key)
}

func (c *Controller) SessionSet(key string, value any) {
	c.sessionHasChanges = true
	c.session.Values[key] = value
}

func (c *Controller) initSession() {
	s, err := Store.Get(c.Request, SessionName)
	if err != nil {
		fmt.Println("Error while reading the session:", err)
	}
	c.session = s
	saver := &sessionSaver2{
		ResponseWriter: c.ResponseWriter,
		c:              c,
	}
	c.ResponseWriter = saver

	c.ViewVar("Session", c.SessionValues())
	c.ViewVar("Flashes", c.FlashesByType())
}

type sessionSaver2 struct {
	http.ResponseWriter
	c     *Controller
	saved bool
}

func (saver *sessionSaver2) WriteHeader(code int) {
	saver.save()
	saver.ResponseWriter.WriteHeader(code)
}
func (saver *sessionSaver2) didSessionChange() bool {
	return saver.c.sessionHasChanges
}

func (saver *sessionSaver2) save() error {
	if saver.saved || !saver.didSessionChange() {
		return nil
	}

	saver.saved = true
	err := saver.c.session.Save(saver.c.Request, saver.ResponseWriter)
	if err != nil {
		panic(err)
	}
	return nil
}

func (saver *sessionSaver2) Write(b []byte) (int, error) {
	saver.save()
	return saver.ResponseWriter.Write(b)
}

// Sesion expire link.
