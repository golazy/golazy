// Package lazyview provides view rendering, helper registration, layouts, and
// render context handling independent from any concrete template engine.
//
// A Views value owns an fs.FS containing templates, the template engines that
// can render those files, and the helpers available during rendering. New
// requires the conventional app layout at "layouts/app.html.tpl" because
// Render defaults to wrapping HTML views in the "app" layout when UseLayout is
// true. Render looks for controller views such as
// "posts/index.html.tpl", falls back to app-wide views such as
// "app/index.html.tpl", and then renders the selected layout with the rendered
// view body available as the "content" variable.
//
// Template engines are opt-in. Import golazy.dev/lazyview/gotmpl, usually as a
// blank import, to register the .tpl engine backed by html/template. Other
// engines can call RegisterEngine for their own file extensions.
//
// Helpers are registered globally with AddHelpers or Helper and may also be
// supplied per render operation with Options.Helpers. A helper whose first
// argument is *Context receives request-local render state after the engine
// binds it for the current render. The built-in "partial" helper renders
// partial templates named with a leading underscore, such as
// "posts/_summary.html.tpl", without a layout.
//
// Fragment is rendered markup that compatible engines can embed without
// treating it as plain text. The gotmpl engine converts Fragment values to
// trusted template content according to their content type. lazyview uses this
// for layout content and partials; lazyapp also uses fragments for cache and
// Turbo helpers.
//
// In a conventional GoLazy application, lazyapp creates one Views value from
// the embedded application views, registers helpers from lazyroutes,
// lazyassets, lazyforms, lazyseo, lazyturbo, cache helpers, and application
// Config.Helpers, then stores the renderer in context through lazycontroller.
// lazycontroller.Base passes route metadata, controller variables, request
// helpers, selected format, variants, and layout choices to lazyview when a
// controller calls Render. lazymailer uses the same renderer from context to
// render mailer templates and layouts without serving an HTTP request.
//
// Use lazyview directly for standalone rendering, custom controller stacks,
// pre-rendered strings, or packages that need the same view filesystem and
// helpers outside lazyapp.
package lazyview
