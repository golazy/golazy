// Package lazypath provides small helpers for generated URL paths.
//
// The package is intentionally smaller than a router. It does not match routes,
// clean paths, reject unsafe base paths, or escape path segments. Callers pass
// the path they already generated, and lazypath only handles the common GoLazy
// convention where the final variadic value may be URLParams. SplitValues
// separates that trailing URLParams value from route parameter values, and
// AppendURLParams encodes those parameters into the query string.
//
// Query values are converted with fmt.Sprint, nil values are skipped, and the
// encoding rules come from net/url.Values. Existing queries are extended with
// "&", and URL fragments stay at the end:
//
//	path := lazypath.AppendURLParams("/posts/hello#comments", lazypath.URLParams{
//		"page": 2,
//		"q":    "go lazy",
//	})
//	// path == "/posts/hello?page=2&q=go+lazy#comments"
//
// lazyroutes uses SplitValues before replacing named route parameters, then
// calls AppendURLParams after it has escaped path parameters with url.PathEscape.
// lazycontroller exposes URLParams as lazycontroller.URLParams and uses this
// package when controller methods call Base.PathFor or Base.MustPathFor. This
// keeps route generation, controller helpers, and standalone path utilities on
// the same query-parameter convention without making lazypath import those
// packages.
package lazypath
