package lazyaction

import (
	"path"
	"reflect"
	"strings"

	"golazy.dev/lazyaction/internal/args"
	"golazy.dev/lazyaction/internal/router"
	"golazy.dev/lazysupport"
)

type ResourceOptions struct {
	ControllerName string // PostsController
	Plural         string // posts
	Singular       string // post
	PathNames      struct{ New, Edit string }
	Path           string // "" means default, "/" means empty
	ParamName      string
}

type Resource struct {
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
	if r.Plural == "" {
		r.Plural = strings.TrimSuffix(lazysupport.Underscorize(r.ControllerName), "_controller") // "posts"
	}

	if r.Singular == "" {
		r.Singular = lazysupport.Singularize(r.Plural)
	}

	if r.Path == "" {
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
func (r *Resource) Routes() []*router.Route {
	routes := []*router.Route{}
	cType := reflect.TypeOf(r.Controller)
	for i := 0; i < cType.NumMethod(); i++ {
		method := cType.Method(i)
		name := method.Name

		ins, outs, err := args.ExtractArgs(method)
		if err != nil {
			panic(err)
		}

		verb, path, methodName := r.analyzeName(name)
		routeName := r.Plural + "#" + lazysupport.Underscorize(methodName)

		verbs := strings.Split(verb, "|")
		for _, v := range verbs {

			route := &router.Route{
				Verb:           v,
				Path:           path,
				Name:           routeName,
				Target:         method.Func,
				Args:           ins,
				Rets:           outs,
				ControllerName: r.ControllerName,
				Plural:         r.Plural,
				Singular:       r.Singular,
				ParamName:      r.ParamName,
				Controller:     r.Controller,
			}

			routes = append(routes, route)
		}

	}
	return routes
}

func (r *Resource) analyzeName(method string) (verb, path, methodName string) {
	pathSegments := make([]string, len(r.Prefix))
	copy(pathSegments, r.Prefix)

	switch method {
	case "Index":
		return "GET", "/" + strings.Join(pathSegments, "/"), method
	case "Create":
		return "POST", "/" + strings.Join(pathSegments, "/"), method
	case "New":
		pathSegments = append(pathSegments, r.PathNames.New)
		return "GET", "/" + strings.Join(pathSegments, "/"), method
	case "Show":
		pathSegments = append(pathSegments, r.ParamName)
		return "GET", "/" + strings.Join(pathSegments, "/"), method
	case "Edit":
		pathSegments = append(pathSegments, r.ParamName)
		pathSegments = append(pathSegments, r.PathNames.Edit)
		return "GET", "/" + strings.Join(pathSegments, "/"), method
	case "Update":
		pathSegments = append(pathSegments, r.ParamName)
		return "PUT|PATCH", "/" + strings.Join(pathSegments, "/"), method
	case "Destroy":
		pathSegments = append(pathSegments, r.ParamName)
		return "DELETE", "/" + strings.Join(pathSegments, "/"), "destroy"
	}
	// Add param if it is a member function
	if strings.HasPrefix(method, Member) {
		pathSegments = append(pathSegments, r.ParamName)
		method = strings.TrimPrefix(method, Member)
	}
	verb, method = prefixes.TrimPrefix(method)
	if method != "" {
		pathSegments = append(pathSegments, lazysupport.Underscorize(method))
	}
	return strings.ToUpper(verb), "/" + strings.Join(pathSegments, "/"), method
}

var prefixes = lazysupport.NewStringSet("Get", "Post", "Delete", "Patch", "Put", "Options")
var Actions = lazysupport.NewStringSet("Index", "Show", "Create", "Update", "Destroy", "New", "Edit")

const Member = "Member"
