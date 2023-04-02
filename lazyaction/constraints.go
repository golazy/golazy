package lazyaction

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyaction/router"
)

type Constraints struct {
	Prefix   string // "" means default, "/" means empty
	Scheme   string
	Domain   string
	Port     string
	resource *Resource

	d *Dispatcher
}

func (c *Constraints) Route(def string, target any) {
	if c.d.router == nil {
		c.d.router = router.NewRouter[Action]()
	}

	path := def
	method := "GET"

	if strings.Contains(def, " ") {
		parts := strings.Split(def, " ")
		method = parts[0]
		path = parts[1]
	}

	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}

	if c.Scheme != "" {
		u.Scheme = c.Scheme
	}

	if c.Prefix != "" {
		if c.Prefix[0] != '/' {
			c.Prefix = "/" + c.Prefix
		}
		if len(c.Prefix) > 2 && c.Prefix[len(c.Prefix)-1] == '/' {
			c.Prefix = c.Prefix[:len(c.Prefix)-1]
		}
		u.Path = c.Prefix + u.Path
	}

	if c.Domain != "" || c.Port != "" {
		host := c.Domain
		if host == "" {
			host = u.Hostname()
		}
		port := c.Port
		if port == "" {
			port = u.Port()
		}
		if port == "" {
			u.Host = host
		} else {
			u.Host = fmt.Sprintf("%s:%s", host, port)
		}
	}

	action := &Action{
		Dispatcher: c.d,
		Verb:       method,
		Assets:     c.d.Assets,
		URL:        *u,
		Name:       "Annonymous",
		Generators: &map[string][]args.Gen{},
	}

	switch t := target.(type) {
	case http.HandlerFunc:
		action.handler = t
	case http.Handler:
		action.handler = t.ServeHTTP
	default:
		if reflect.ValueOf(t).Kind() == reflect.Func {
			action.fn = args.NewFn(t)
			break
		}
		panic(fmt.Sprintf("invalid argument type: %T", t))
	}

	c.d.router.Add(def, action)
}

func (c *Constraints) updateAction(action *Action) {
	action.Dispatcher = c.d
	action.Assets = c.d.Assets
	u := &action.URL

	if c.Scheme != "" {
		u.Scheme = c.Scheme
	}

	if c.Prefix != "" {
		if c.Prefix[0] != '/' {
			c.Prefix = "/" + c.Prefix
		}
		if len(c.Prefix) > 2 && c.Prefix[len(c.Prefix)-1] == '/' {
			c.Prefix = c.Prefix[:len(c.Prefix)-1]
		}
		u.Path = c.Prefix + u.Path
	}

	if c.Domain != "" || c.Port != "" {
		host := c.Domain
		if host == "" {
			host = u.Hostname()
		}
		port := c.Port
		if port == "" {
			port = u.Port()
		}
		if port == "" {
			u.Host = host
		} else {
			u.Host = fmt.Sprintf("%s:%s", host, port)
		}
	}

}
func (c *Constraints) Memeber() *Constraints {
	prefix := "/:id"
	if c.resource != nil {
		prefix = "/" + c.resource.ParamName
	}
	c2 := *c
	c2.Prefix = c.Prefix + prefix
	return &c2
}

func (c *Constraints) Resource(target any, options ...*ResourceOptions) *Constraints {
	if c.d.router == nil {
		c.d.router = router.NewRouter[Action]()
	}
	if len(options) == 0 {
		options = append(options, &ResourceOptions{})
	}
	resource, err := newResource(target, options[0])
	if err != nil {
		panic(err)
	}
	for _, action := range resource.Actions() {
		c.updateAction(action)
		c.d.router.Add(action.Verb+" "+action.URL.String(), action)
	}

	c2 := *c
	c2.resource = resource
	c2.Prefix = c.Prefix + "/" + resource.Path
	return &c2
}
