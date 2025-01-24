package lazycontroller

import "golazy.dev/lazysupport"

func (c *Base) ensureCSRFToken() {
	v := c.SessionGet("csrf_token")
	if v.IsOk() {
		c.csrf = v.String()
	} else {
		c.csrf = lazysupport.RandomString(20)
		c.SessionSet("csrf_token", c.csrf)
	}
	c.ViewVar("csrf_token", c.csrf)
	if c.R.Method == "GET" {
		return
	}
	rtoken := c.R.Header.Get("X-Csrf-Token")
	if rtoken != c.csrf {
		// Remove session
		panic("CSRF token mismatch")
	}
}

func (c *Base) CSRFToken() string {
	return c.csrf

}
