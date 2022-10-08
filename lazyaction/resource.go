package lazyaction

import (
	"reflect"
	"strings"

	"github.com/golazy/golazy/lazysupport"
)

func Resource(Controller interface{}) Routes {
	cType := reflect.TypeOf(Controller)
	fullName := cType.Elem().Name()
	name := lazysupport.ToSnakeCase(fullName)
	name = strings.TrimSuffix(name, "_controller")
	param_name := lazysupport.Singularize(name) + "_id"

	memberPrefix := CatchAllPrefix{
		ParamName: param_name,
	}
	prefix := Prefix{
		Prefix: name,
	}

	addCollectionRedirect := false
	addMemeberRedirect := true

	routes := Routes{}

	for i := 0; i < cType.NumMethod(); i++ {
		m := cType.Method(i)
		a := action{
			c:        Controller,
			fullName: fullName,
			name:     name,
			method:   m.Name,
		}

		dest := fullName + "#" + m.Name

		switch m.Name {
		case "Index":
			routes = append(routes, Path{name, "GET", dest, a})
			addCollectionRedirect = true
		case "New":
			prefix.Routes = append(prefix.Routes, Path{"new", "GET", dest, a})
		case "Create":
			routes = append(routes, Path{name, "POST", dest, a})
			addCollectionRedirect = true
		case "Show":
			prefix.Routes = append(prefix.Routes, CatchAllPath{param_name, "GET", dest, a})
			addMemeberRedirect = true
		case "Update":
			prefix.Routes = append(prefix.Routes, CatchAllPath{param_name, "PUT", dest, a})
			prefix.Routes = append(prefix.Routes, CatchAllPath{param_name, "PATCH", dest, a})
			addMemeberRedirect = true
		case "Delete":
			prefix.Routes = append(prefix.Routes, CatchAllPath{param_name, "DELETE", dest, a})
			addMemeberRedirect = true
		default:
			found, forCollection, path, method := getRouteInfoFromMethodName(m.Name)
			if !found {
				continue
			}
			if forCollection {
				memberPrefix.Routes = append(memberPrefix.Routes, Path{path, method, dest, a})
				continue
			}
			prefix.Routes = append(prefix.Routes, Path{path, method, dest, a})
		}
	}

	if addCollectionRedirect {
		prefix.Routes = append(prefix.Routes, RedirectPath{To: "."})
	}
	if addMemeberRedirect {
		memberPrefix.Routes = append(memberPrefix.Routes, RedirectPath{To: "."})
	}

	if len(memberPrefix.Routes) > 0 {
		prefix.Routes = append(prefix.Routes, memberPrefix)
	}
	if len(prefix.Routes) > 0 {
		routes = append(routes, prefix)
	}
	return routes
}

var prefixes = []string{"Get", "Post", "Delete", "Patch", "Put"}
var memberPrefix = "For"

func getVerbFromMethod(name string) (found bool, verb, rest string) {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true, strings.ToUpper(prefix), strings.TrimPrefix(name, prefix)
		}
	}
	return false, "", ""
}

func getCollectionFromMethod(name string) (forCollection bool, rest string) {
	if strings.HasPrefix(name, "Member") {
		return true, strings.TrimPrefix(name, "Member")
	}
	return false, name
}

func getRouteInfoFromMethodName(name string) (found, forCollection bool, path, method string) {
	path = name
	method = ""
	forCollection, name = getCollectionFromMethod(name)

	found, method, name = getVerbFromMethod(name)
	if !found {
		return false, false, "", ""
	}

	path = lazysupport.ToSnakeCase(name)

	return
}

type action struct {
	c        interface{}
	fullName string
	name     string
	method   string
}

func (a action) ServeHTTP(w ResponseWriter, r *Request) {
	inputs := []reflect.Value{
		reflect.ValueOf(w),
		reflect.ValueOf(r),
	}

	reflect.ValueOf(a.c).MethodByName(a.method).Call(inputs)
}
