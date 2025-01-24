package lazycontroller

func (c *Base) HeaderSet(key string, value string) {
	c.W.Header().Set(key, value)
}
func (c *Base) HeaderGet(key string, value string) {
	c.R.Header.Get(key)
}
