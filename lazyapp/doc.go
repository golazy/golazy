// Package lazyapp composes lower-level GoLazy packages into a runnable web
// application.
//
// Most applications use New at the application boundary. New creates the
// application context, initializes dependencies, opens configured views, creates
// the lazycontroller renderer, builds a lazyroutes scope, calls the route drawer,
// registers framework and application helpers with lazyview, initializes cache,
// sessions, jobs, robots.txt, sitemap endpoints, and optional control-plane
// handlers, then returns one http.Handler.
//
// Public files and generated assets are registered with lazyassets and mounted
// as the final fallback after dynamic routes. View helpers from lazyroutes,
// lazyassets, lazyforms, lazyseo, lazyturbo, cache helpers, and Config.Helpers
// are passed to the lazyview renderer before templates are cached. Controllers
// usually embed lazycontroller.Base; lazyroutes binds each request to that base,
// and lazycontroller renders through the renderer created here.
//
// Direct package use still makes sense when an application needs only one
// layer: use lazyroutes for a standalone route table, lazyassets for standalone
// hashed asset serving, lazyview for template rendering without controllers, or
// lazycontroller when a custom application shell wants GoLazy controller
// rendering without the rest of this composition. Use lazyapp when those pieces
// should behave like a conventional GoLazy application.
//
// For embedded application files, pass subdirectories to Config with MustSub:
//
//	//go:embed public views
//	var files embed.FS
//
//	app := lazyapp.New(lazyapp.Config{
//		Public: lazyapp.MustSub(files, "public"),
//		Views:  lazyapp.MustSub(files, "views"),
//	})
package lazyapp
