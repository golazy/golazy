// Package lazyseo renders common document metadata for GoLazy views and other
// lazyview renderers.
//
// The package owns the metadata values that normally appear in an HTML
// document head: the page title, description, author, language, canonical URL,
// alternate URLs, social sharing tags, article timestamps, and schema.org
// JSON-LD. A canonical URL is the preferred permanent URL for the current
// content; use it when the same page can be reached through more than one path
// or query string. Alternate URLs describe other language, media, or format
// versions. JSON-LD is structured data encoded as a
// <script type="application/ld+json"> element so search and social crawlers can
// understand the page as an article, product, organization, or another schema
// type.
//
// lazyapp installs Helpers automatically after it creates the lazyview renderer.
// The application-wide defaults come from lazyapp.Config.SEO, and those
// defaults are merged with request-local metadata before the helpers render.
// Layouts can call {{seo}} inside <head> and {{seo_lang}} on the html element.
// lazycontroller.Base exposes SEO, Title, Description, Canonical, Alternate,
// Kind, OpenGraph, TwitterCard, JSONLD, and related convenience methods that
// store request-local metadata for those helpers. lazycontroller.Base.Metadata
// can also read small metadata methods from a model and fill lazyseo.Meta.
//
// Use PageKind values such as Article or WebPage when one choice should set
// both Open Graph and schema.org names. Use OpenGraphType, SchemaType, and the
// OpenGraph or TwitterCard structs when a crawler-specific escape hatch is
// needed. The lazyseo/jsonld subpackage contains small schema.org value types
// for common JSON-LD payloads, and DefaultJSONLD builds a conventional payload
// from a Meta value when lazycontroller.Metadata has enough information.
//
// lazyseo does not generate robots.txt or sitemap.xml files. Robots directives
// tell crawlers which paths should be crawled, and sitemaps list canonical URLs
// that should be discovered; those are site-level documents owned by an
// application or a future routing/indexing package. This package only renders
// metadata for the current view.
//
// The package can also be used directly with any value that supports Set(string,
// any), or by passing a "seo" variable to lazyview. That keeps metadata
// rendering independent from controller internals and from a specific template
// engine.
package lazyseo
