package lazycontroller

import "golazy.dev/lazysupport"

func (c *Controller) ensureCSRFToken() {
	v := c.SessionGet("csrf_token")
	if v.IsOk() {
		c.csrf = v.String()
	} else {
		c.csrf = lazysupport.RandomString(20)
		c.SessionSet("csrf_token", c.csrf)
	}
	c.ViewVar("csrf_token", c.csrf)
	if c.Request.Method == "GET" {
		return
	}
	rtoken := c.Request.Header.Get("X-Csrf-Token")
	if rtoken != c.csrf {
		// Remove session
		panic("CSRF token mismatch")
	}
}

func (c *Controller) CSRFToken() string {
	return c.csrf

}
