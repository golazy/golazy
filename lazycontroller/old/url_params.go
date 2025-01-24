package lazycontroller

import (
	"strconv"
	"strings"
)

func (c *Controller) URLParam(s string) (string, bool) {
	components := strings.Split(c.route.Path, "/")
	urlComponents := strings.Split(c.Request.URL.Path, "/")
	for i, c := range components {
		if strings.HasPrefix(c, ":"+s) {
			return urlComponents[i], true
		}
	}
	return "", false
}

func (c *Controller) URLParamInt(s string) (int, bool) {
	v, ok := c.URLParam(s)
	if !ok {
		return 0, false
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return i, true
}

func (c *Controller) URLParamUint(s string) (uint, bool) {
	v, ok := c.URLParam(s)
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(i), true
}
