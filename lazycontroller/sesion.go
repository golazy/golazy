package lazycontroller

import (
	"fmt"
	"net/http"

	"golazy.dev/lazysupport"

	"github.com/gorilla/sessions"
)

// TODO: Add configuration for the session Store including the session name
var Store = sessions.NewCookieStore([]byte("Tjz@-QNkekP2KH8oF9A_GNssbwvftXqv"))
var SessionName = "lazy_session"

func (c *Base) SessionGet(key string) *lazysupport.Value {
	value, ok := c.session.Values[key]
	if !ok {
		return lazysupport.NewValue(nil)
	}

	return lazysupport.NewValue(value)
}

// SessionValues return all values in the session.
// Modifying the returned map may or may not work.
// Use SessionSet or SessionDelete instead.
func (c *Base) SessionValues() map[any]any {
	return c.session.Values
}

func (c *Base) SessionDelete(key string) {
	c.sessionHasChanges = true
	delete(c.session.Values, key)
}

func (c *Base) SessionSet(key string, value any) {
	c.sessionHasChanges = true
	c.session.Values[key] = value
}

func (c *Base) initSession() {
	s, err := Store.Get(c.R, SessionName)
	if err != nil {
		fmt.Println("Error while reading the session:", err)
	}
	c.session = s
	saver := &sessionSaver2{
		ResponseWriter: c.W,
		c:              c,
	}
	c.W = saver

	c.ViewVar("Session", c.SessionValues())
	c.ViewVar("Flashes", c.FlashesByType())
}

type sessionSaver2 struct {
	http.ResponseWriter
	c     *Base
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
	err := saver.c.session.Save(saver.c.R, saver.ResponseWriter)
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
