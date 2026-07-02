// Package lazyroutes provides the GoLazy route scope, named route helpers, REST
// resources, request route metadata, and controller action binding.
//
// In a normal app, package lazyapp creates the root Scope, passes it to the
// application's Draw function, stores Scope.PathFor in the application context
// for lazycontroller.Base.PathFor, and passes Scope.RegisterHelpers to
// lazyview. Those helpers make path_for, link_to, attr, data, and
// unless_current available in templates. lazyforms also consumes the router's
// PathFor and PathForModel methods when form helpers build resource actions.
//
// A Scope is also an http.Handler. It wraps an http.ServeMux, stores the
// registered RouteTable, and attaches matched Route metadata plus named
// parameter values to each request context. RouteFromRequest and
// RouteFromContext read that metadata for lower-level integrations.
//
// # Drawing routes
//
// HandleFunc registers a route without a controller. Get, Post, Put, Patch,
// and Delete register a controller constructor and action. The constructor must
// have the form:
//
//	func(context.Context) (*PostsController, error)
//
// Standard actions may be methods such as:
//
//	func (c *PostsController) Show(http.ResponseWriter, *http.Request) error
//
// Actions can also ask for request-derived inputs. The action planner resolves
// http.ResponseWriter, *http.Request, context.Context, route parameters
// converted to strings, ints, or slices, and values returned by controller
// methods named GenX. Generator methods may depend on other generated values,
// and their errors are handled through lazycontroller's error path.
//
// Resources registers conventional REST routes for controller methods named
// Index, New, Create, Show, Edit, Update, and Delete when those methods exist.
// Resource custom routes add collection or member actions, nested resources
// build paths under the parent member path, and Model records the create,
// update, and delete route names used by PathForModel.
//
// Namespace prefixes the URL path, route name, and view namespace. Path prefixes
// only the URL path. As prefixes only route names. The namespace value is passed
// through lazycontroller to lazyview, where it changes view and layout lookup
// directories for namespaced controllers.
//
// # Paths and helpers
//
// PathFor builds URLs from route names and path parameter values. Path
// parameters are escaped, and lazypath.URLParams values are appended as query
// parameters. Route names are inferred from the route path when no explicit
// name is provided: "/" becomes "root", "/posts/{post_id}" becomes "posts",
// and resource routes use conventional names such as "posts", "post",
// "new_post", and "edit_post". RegisterHelpers exposes PathFor and safe link
// helpers to lazyview templates; lazyapp calls it automatically.
//
// The router also understands registered format suffixes from lazycontroller.
// For example, a request to "/posts/1.json" can match a route registered as
// "/posts/{post_id}"; the request path passed to the handler is stripped to the
// logical route path and the format is stored in context for
// lazycontroller.FormatFromRequest.
//
// For readable routes, trailing slash requests are canonicalized before route
// dispatch. GET and HEAD requests redirect permanently to the same route without
// trailing slashes when that slashless path is registered; unknown paths,
// public assets, and unsafe methods are left alone.
//
// The package can be used without lazyapp for small HTTP services, tests, or
// custom application shells:
//
//	router := lazyroutes.New(context.Background())
//	router.HandleFunc(http.MethodGet, "/health", func(w http.ResponseWriter, r *http.Request) error {
//		_, err := w.Write([]byte("ok"))
//		return err
//	})
//	http.ListenAndServe(":3000", router)
//
// For a full GoLazy application, lazyapp layers lazydispatch around the router,
// adds public-file and asset fallbacks, installs lazycontroller rendering, and
// registers framework helpers before templates are cached.
//
// # Development hooks
//
// RegisterLazyDevHandlers exists only in files built with the lazydev build
// tag. lazyapp registers it on the lazycontrolplane.ControlPlane in development
// builds. The current endpoint is GET /routes, which returns the registered
// RouteTable as JSON for the development panel; production builds do not expose
// that route.
package lazyroutes
