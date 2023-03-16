package lazyaction

import (
	"net/url"
	"path"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazysupport"
)

type ResourceOptions struct {
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	PathNames      struct{ New, Edit string }
	Path           string // "" means default, "/" means empty
	Scheme         string
	Domain         string
	Port           string
	ParamName      string
	Name           string
}

type Resource struct {
	BaseUrl url.URL
	Layout  *args.Fn
	ResourceOptions
	Controller any
	Prefix     []string
}

func newResource(controller any, opts *ResourceOptions) (*Resource, error) {

	r := &Resource{
		Controller: controller,
	}

	if opts != nil {
		r.ResourceOptions = *opts
	}

	if r.PathNames.New == "" {
		r.PathNames.New = "new"
	}

	if r.PathNames.Edit == "" {
		r.PathNames.Edit = "edit"
	}

	cType := reflect.TypeOf(r.Controller)
	if r.ControllerName == "" {
		r.ControllerName = cType.Elem().Name() // "PostsController"
	}
	if r.ControllerName == "Controller" {
		n := path.Base(cType.Elem().PkgPath())
		r.ControllerName = lazysupport.Camelize(n) + "Controller"
	}

	if r.Name == "" {
		r.Name = strings.TrimSuffix(r.ControllerName, "Controller") // "Posts"
	}

	if r.Plural == "" {
		r.Plural = lazysupport.Underscorize(r.Name) // "posts"
	}

	if r.Singular == "" {
		r.Singular = lazysupport.Singularize(lazysupport.Underscorize(r.Name))
	}

	if r.Path == "" {
		r.Path = "/"
		r.Prefix = []string{r.Plural}
	} else if r.Path != "/" {
		r.Prefix = strings.Split(strings.TrimPrefix(r.Path, "/"), "/")
	}

	if r.ParamName == "" {
		r.ParamName = ":" + r.Singular + "_id" // :post_id
	} else {
		r.ParamName = ":" + strings.TrimPrefix(r.ParamName, ":")
	}

	return r, nil
}

//	func (r *Resource) addSubResources() {
//		for _, sr := range r.SubResources {
//			resource := NewResource(sr)
//			for _, action := range resource.ResourceActions {
//				a := *action
//				segments := make([]string, len(r.Prefix))
//				copy(segments, r.Prefix)
//				segments = append(segments, r.ParamName)
//
//				a.Path = "/" + strings.Join(segments, "/") + a.Path
//				a.RouteName = r.Singular + "_" + a.ResourceName
//				if a.ActionName != "" {
//					a.RouteName = a.ActionName + "_" + a.RouteName
//				}
//
//				for i := range a.ParamsPosition {
//					a.ParamsPosition[i] += len(segments)
//				}
//
//				a.ParamsPosition = append([]int{len(segments) - 1}, a.ParamsPosition...)
//
//				r.ResourceActions = append(r.ResourceActions, &a)
//			}
//		}
//	}
func (r *Resource) Actions() []*Action {
	actions := []*Action{}
	cType := reflect.TypeOf(r.Controller)
	cVal := reflect.ValueOf(r.Controller)
	generators := make(map[string][]args.Gen)

	for i := 0; i < cType.NumMethod(); i++ {
		methodT := cType.Method(i)
		methodV := cVal.Method(i)
		name := methodT.Name
		switch {
		case isAction(name):
			actions = append(actions, r.genActionsForMethodN(cVal, i)...)
		case strings.HasPrefix(name, "Gen"):
			fn := args.NewGen((cVal.Method(i)))
			t := fn.Outs[0]
			generators[t] = append(generators[t], fn)
		case name == "RenderLayout":
			r.Layout = args.NewFn(methodV)
		}
	}

	// fill actions with the generator
	for _, action := range actions {
		action.Generators = &generators
		action.Layout = r.Layout
	}

	return actions
}

func (r *Resource) genActionsForMethodN(val reflect.Value, i int) []*Action {
	routes := []*Action{}
	meth := val.Method(i)
	cType := reflect.TypeOf(r.Controller)
	methodT := cType.Method(i)

	name := methodT.Name

	verb, path, methodName, paramName := r.analyzeName(name)

	routeName := lazysupport.Underscorize(r.Name) + "#" + lazysupport.Underscorize(methodName)

	u, err := url.Parse(path)
	if err != nil {
		panic(err)
	}
	u.Scheme = r.Scheme
	u.Host = r.Domain
	if r.Port != "" {
		u.Host = u.Host + ":" + r.Port
	}

	fn := args.NewFn(meth)

	route := &Action{
		Verb:    verb,
		Method:  name,
		methodI: i,
		URL:     *u,
		Name:    routeName,
		ins:     fn.Ins,
		outs:    fn.Outs,

		ControllerName: r.ControllerName,
		Plural:         r.Plural,
		Singular:       r.Singular,
		ParamName:      paramName,
		Controller:     r.Controller,
	}

	routes = append(routes, route)

	return routes

}

func isAction(name string) bool {
	switch name {
	case "Index", "Create", "New", "Show", "Edit", "Update", "Destroy":
		return true
	}
	name = strings.TrimPrefix(name, Member)
	return prefixes.HasPrefix(name)
}

func (r *Resource) analyzeName(method string) (verb, path, methodName, paramName string) {
	pathSegments := make([]string, len(r.Prefix))
	copy(pathSegments, r.Prefix)

	switch method {
	case "Index":
		return "GET", "/" + strings.Join(pathSegments, "/"), method, ""
	case "Create":
		return "POST", "/" + strings.Join(pathSegments, "/"), method, ""
	case "New":
		pathSegments = append(pathSegments, r.PathNames.New)
		return "GET", "/" + strings.Join(pathSegments, "/"), method, ""
	case "Show":
		pathSegments = append(pathSegments, r.ParamName)
		return "GET", "/" + strings.Join(pathSegments, "/"), method, r.ParamName
	case "Edit":
		pathSegments = append(pathSegments, r.ParamName)
		pathSegments = append(pathSegments, r.PathNames.Edit)
		return "GET", "/" + strings.Join(pathSegments, "/"), method, r.ParamName
	case "Update":
		pathSegments = append(pathSegments, r.ParamName)
		return "PUT,PATCH", "/" + strings.Join(pathSegments, "/"), method, r.ParamName
	case "Destroy":
		pathSegments = append(pathSegments, r.ParamName)
		return "DELETE", "/" + strings.Join(pathSegments, "/"), "destroy", r.ParamName
	}
	// Add param if it is a member function
	if strings.HasPrefix(method, Member) {
		pathSegments = append(pathSegments, r.ParamName)
		method = strings.TrimPrefix(method, Member)
		paramName = r.ParamName
	}
	verb, method = prefixes.TrimPrefix(method)
	if method != "" {
		pathSegments = append(pathSegments, lazysupport.Underscorize(method))
	}
	if verb == "" {
		verb = "GET"
	}
	return strings.ToUpper(verb), "/" + strings.Join(pathSegments, "/"), method, paramName
}

var prefixes = lazysupport.NewStringSet("Get", "Post", "Delete", "Patch", "Put", "Options")
var DefaultActions = lazysupport.NewStringSet("Index", "Show", "Create", "Update", "Destroy", "New", "Edit")

const Member = "Member"
