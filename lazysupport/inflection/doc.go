// Package inflection contains word and naming helpers used by GoLazy
// conventions.
//
// lazyroutes uses these helpers to infer REST resource names. Applications can
// use the package directly when they need the same singular, plural, or naming
// behavior without importing the router. Applications with domain-specific
// words can register Irregular rules during package initialization before
// routes or other conventional names are derived.
package inflection
