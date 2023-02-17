package lazyaction

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	"golazy.dev/lazysupport"
)

func NewResource(rd *ResourceDefinition) *Resource {
	r := &Resource{
		ResourceDefinition: *rd,
		Actions:            []*Action{},
	}

	r.setDefaults()
	r.analyzeMethods()
	r.addSubResources()
	return r
}

type Resource struct {
	ResourceDefinition
	Prefix  []string
	Actions []*Action
}

func (r *Resource) addSubResources() {
	for _, sr := range r.SubResources {
		resource := NewResource(sr)
		for _, action := range resource.Actions {
			a := *action
			segments := make([]string, len(r.Prefix))
			copy(segments, r.Prefix)
			segments = append(segments, r.ParamName)

			a.Path = "/" + strings.Join(segments, "/") + a.Path
			a.RouteName = r.Singular + "_" + a.ResourceName
			if a.ActionName != "" {
				a.RouteName = a.ActionName + "_" + a.RouteName
			}

			for i := range a.ParamsPosition {
				a.ParamsPosition[i] += len(segments)
			}

			a.ParamsPosition = append([]int{len(segments) - 1}, a.ParamsPosition...)

			r.Actions = append(r.Actions, &a)
		}
	}
}

func (r *Resource) analyzeMethods() {
	cType := reflect.TypeOf(r.Controller)
	for i := 0; i < cType.NumMethod(); i++ {
		method := cType.Method(i)
		switch {
		case IsRouterMethod(method.Name):
			r.Actions = append(r.Actions, NewAction(method.Name, r))
		}
	}
}

func (r *Resource) setDefaults() {
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

}

func (r *Resource) pathForMethod(method string) (string, []int) {
	argsPos := []int{}

	pathSegments := make([]string, len(r.Prefix))
	copy(pathSegments, r.Prefix)

	switch method {
	case "Index":
	case "Create":
	case "New":
		pathSegments = append(pathSegments, r.PathNames.New)
	case "Show":
		argsPos = append(argsPos, len(pathSegments))
		pathSegments = append(pathSegments, r.ParamName)
	case "Edit":
		argsPos = append(argsPos, len(pathSegments))
		pathSegments = append(pathSegments, r.ParamName)
		pathSegments = append(pathSegments, r.PathNames.Edit)
	case "Update":
		argsPos = append(argsPos, len(pathSegments))
		pathSegments = append(pathSegments, r.ParamName)
	case "Destroy":
		argsPos = append(argsPos, len(pathSegments))
		pathSegments = append(pathSegments, r.ParamName)
	default:
		// Handle prefixes like Get, Post, Delete, Patch, Put, Options alone
		if strings.HasPrefix(method, Member) {
			argsPos = append(argsPos, len(pathSegments))
			pathSegments = append(pathSegments, r.ParamName)
			method = strings.TrimPrefix(method, Member)
		}
		_, method := prefixes.TrimPrefix(method)
		if method != "" {
			pathSegments = append(pathSegments, lazysupport.Underscorize(method))
		}

	}

	return "/" + strings.Join(pathSegments, "/"), argsPos
}

func (r *Resource) verbForMethod(method string) string {
	verb := "GET"
	switch method {
	case "Index", "Edit", "New", "Show":
	case "Create":
		verb = "POST"
	case "Update":
		verb = "PUT|PATCH"
	case "Destroy":
		verb = "DELETE"
	default:
		method = strings.TrimPrefix(method, Member)
		verb, _ = prefixes.TrimPrefix(method)
		if verb != "" {
			verb = strings.ToUpper(verb)
		}
	}

	return verb
}

func (r *Resource) nameForMethod(method string) (resourceName, action string) {
	resourceName = r.Singular
	switch method {
	case "Index", "Create":
		resourceName = r.Plural
	case "Destroy", "Update", "Show":
	case "New":
		action = "new"
	case "Edit":
		action = "edit"
	default:
		method = strings.TrimPrefix(method, Member)
		_, name := prefixes.TrimPrefix(method)
		action = lazysupport.Underscorize(name)
	}
	return
}

func (r *Resource) Routes() string {
	t := lazysupport.Table{
		Header: []string{"Verb", "Name", "Path", "Destination"},
		Values: [][]string{},
	}
	for _, r := range r.Actions {
		t.Values = append(t.Values, []string{
			r.RouteName,
			r.Verb,
			r.Path,
			r.Destination,
			fmt.Sprintf("%+v", r.ParamsPosition),
		})
	}

	return t.String()
}

var prefixes = lazysupport.NewStringSet("Get", "Post", "Delete", "Patch", "Put", "Options")
var Actions = lazysupport.NewStringSet("Index", "Show", "Create", "Update", "Destroy", "New", "Edit")

const Member = "Member"

func IsRouterMethod(method string) bool {
	if Actions.Has(method) {
		return true
	}
	if prefixes.HasPrefix(method) {
		return true
	}
	if strings.HasPrefix(method, Member) {
		return IsRouterMethod(strings.TrimPrefix(method, Member))
	}
	return false
}
