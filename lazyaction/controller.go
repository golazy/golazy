package lazyaction

import (
	"reflect"
	"strings"

	"github.com/golazy/golazy/lazysupport"
)

type Controller struct {
	target    interface{}
	cType     reflect.Type
	fullName  string
	name      string
	paramName string
	actions   []*action
}

func newController(target interface{}) *Controller {
	c := &Controller{
		target: target,
	}
	c.cType = reflect.TypeOf(c.target)
	c.fullName = c.cType.Elem().Name()
	c.name = strings.TrimSuffix(lazysupport.Underscorize(c.fullName), "_controller")
	c.paramName = ":" + lazysupport.Singularize(c.name) + "_id"

	c.actions = []*action{}
	for i := 0; i < c.cType.NumMethod(); i++ {
		a := c.actionForMethod(c.cType.Method(i))
		if a != nil {
			c.actions = append(c.actions, a)
		}
	}

	return c
}

func (c Controller) Routes() []*action {
	return c.actions
}

func (c *Controller) actionForMethod(m reflect.Method) *action {
	a := &action{
		Controller:    c,
		TopController: c.target,
		Destination:   c.fullName + "#" + m.Name,
		Function:      m.Name,
		Verb:          "GET",
		Method:        reflect.ValueOf(c.target).MethodByName(m.Name),
		Args:          []string{},
	}

	switch m.Name {
	case "Index":
	case "New":
		a.Path = "new"
	case "Create":
		a.Verb = "POST"
	case "Show":
		a.Path = c.paramName
	case "Update":
		a.Path = c.paramName
		a.Verb = "PUT|PATCH"
	case "Delete":
		a.Path = c.paramName
		a.Verb = "DELETE"
	default:
		var found, member bool
		var path string
		found, member, path, a.Verb = getRouteInfoFromMethodName(m.Name)
		if !found {
			return nil
		}
		if member {
			a.Path = c.paramName + "/" + path
		} else {
			a.Path = path
		}
	}

	// Get method Arguments
	methodT := a.Method.Type()
	for i := 0; i < methodT.NumIn(); i++ {
		a.Args = append(a.Args, methodT.In(i).String())
	}
	// Get method return
	for i := 0; i < methodT.NumOut(); i++ {
		a.Returns = append(a.Returns, methodT.Out(i).String())
	}

	return a

}

var prefixes = []string{"Get", "Post", "Delete", "Patch", "Put"}

func getRouteInfoFromMethodName(name string) (found, member bool, path, method string) {
	found = false

	member = strings.HasPrefix(name, "Member")
	if member {
		name = strings.TrimPrefix(name, "Member")
	}

	for _, m := range prefixes {
		if strings.HasPrefix(name, m) {
			method = strings.ToUpper(m)
			name = strings.TrimPrefix(name, m)
			break
		}
	}
	path = lazysupport.Underscorize(name)
	found = method != ""
	return
}
