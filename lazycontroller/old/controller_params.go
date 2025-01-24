package lazycontroller

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"golazy.dev/lazydispatch/values"
)

func (c *Controller) FillWithParams(model string, data any) error {
	ct := c.Request.Header.Get("Content-Type")

	if strings.HasPrefix(ct, "multipart/form-data") {
		err := c.Request.ParseMultipartForm(MaxFormSize)
		if err != nil {
			return err
		}
	} else {
		err := c.Request.ParseForm()
		if err != nil {
			return err
		}
	}

	return values.Values(c.Request.Form).Extract(model).Load(data)
}

func (c *Controller) WithParamBool(name string, fn func(bool)) {
	if c.ParamsIsPresent(name) && slices.Contains(boolValues, c.ParamsGet(name)) {
		fn(c.ParamsGetBool(name))
	}
}
func (c *Controller) WithParam(name string, fn func(string)) {
	if c.ParamsIsPresent(name) {
		fn(c.ParamsGet(name))
	}
}

func (c *Controller) WithParamInt(name string, fn func(int)) {
	if c.ParamsIsPresent(name) {
		fn(c.ParamsGetInt(name))
	}
}
func (c *Controller) WithParamUint(name string, fn func(uint)) {
	if c.ParamsIsPresent(name) {
		fn(c.ParamsGetUint(name))
	}
}

func (c *Controller) ParamsIsPresent(name string) bool {
	c.Request.ParseForm()
	_, ok := c.Request.Form[name]

	return ok
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB

)

func (c *Controller) ParamsGet(name string) string {
	if c.Request.Form == nil {
		c.Request.ParseMultipartForm(defaultMaxMemory)
	}
	values := c.Request.Form[name]
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func (c *Controller) ParamsGetInt(name string) int {
	v := c.ParamsGet(name)
	if v == "" {
		return 0
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0
	}
	return int(i)
}

func (c *Controller) ParamsGetUint(name string) uint {
	v := c.ParamsGet(name)
	if v == "" {
		return 0
	}
	i, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint(i)
}

var (
	trueValues  = []string{"1", "true", "on", "yes", "y", "t"}
	falseValues = []string{"0", "false", "off", "no", "n", "f"}
	boolValues  = append(trueValues, falseValues...)
)

func (c *Controller) ParamsGetBool(name string) bool {
	v := c.ParamsGet(name)
	if v == "" {
		return false
	}
	if slices.Contains(trueValues, v) {
		return true
	}
	if slices.Contains(falseValues, v) {
		return false
	}
	panic(fmt.Errorf("parameter %s expect to be a boolean, got %q", name, v))
}
