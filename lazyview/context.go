package lazyview

import (
	"context"
	"net/http"
)

// Route is request route metadata made available to renderers and helpers.
type Route struct {
	Name       string
	Method     string
	Path       string
	Namespace  string
	Controller string
	Action     string
	Params     map[string]string
}

// Fragment is rendered output that can be embedded by compatible engines.
type Fragment struct {
	Body        string
	ContentType string
}

// Context contains the request-local state used while rendering a view.
type Context struct {
	Context context.Context
	Request *http.Request

	Views *Views
	Route Route

	Variables map[string]any
	// Data is the value used as dot while executing the current template.
	Data    any
	helpers map[string]any

	Namespace  string
	Controller string
	Action     string
	Partial    string
	Format     string
	Layout     string
}

// HelperFuncs returns helper functions bound to the current render context.
func (c *Context) HelperFuncs() map[string]any {
	helpers := make(map[string]any, len(c.helpers))
	for name, helper := range c.helpers {
		helpers[name] = bindHelper(c, helper)
	}
	return helpers
}

// Helpers returns a copy of the unbound helpers for nested render operations.
func (c *Context) Helpers() map[string]any {
	return copyHelpers(c.helpers)
}

// Helper returns one helper bound to the current render context.
func (c *Context) Helper(name string) (any, bool) {
	helper, ok := c.helpers[name]
	if !ok {
		return nil, false
	}
	return bindHelper(c, helper), true
}
