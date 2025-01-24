package lazycontroller

func (c *Controller) HeaderSet(key string, value string) {
	c.ResponseWriter.Header().Set(key, value)
}
