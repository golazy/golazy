package lazyaction

import (
	"sort"
	"strings"

	"github.com/golazy/golazy/lazysupport"
)

func (r Routes) String() string {

	t := lazysupport.Table{
		Header: []string{"METHOD", "PATH", "DESTINATION"},
		Values: [][]string{},
	}

	r.eachRoute("", func(method, path, destination string) {
		t.Values = append(t.Values, []string{method, path, destination})
	})

	// Sort routes
	sort.Sort(byPath(t.Values))

	return t.String()
}

type byPath [][]string

func (a byPath) Len() int      { return len(a) }
func (a byPath) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byPath) Less(i, j int) bool {
	// : goes before A in the asci table. Replacing : for | (that goes after Z) display memer methods at the end
	alpha := strings.Compare(
		strings.ReplaceAll(a[i][1], ":", "|"),
		strings.ReplaceAll(a[j][1], ":", "|"),
	)
	if alpha != 0 {
		return alpha < 0
	}
	return false
}

func sanitize(method, path, destination string) (string, string, string) {
	if method == "" {
		method = "GET"
	}
	if destination == "" {
		destination = "Annonymous Handler"
	}

	return method, path, destination
}

func (r Routes) eachRoute(prefix string, fn func(method, path, target string)) {

	for _, route := range r {

		switch route := route.(type) {
		case Path:
			fn(sanitize(route.Method, prefix+"/"+route.Path, route.Destination))
		case CatchAllPath:
			fn(sanitize(route.Method, prefix+"/:"+route.ParamName, route.Destination))
		case Prefix:
			route.Routes.eachRoute(prefix+"/"+route.Prefix, fn)
		case CatchAllPrefix:
			route.Routes.eachRoute(prefix+"/:"+route.ParamName, fn)
		}
	}
}
