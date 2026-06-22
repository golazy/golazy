// Package lazypath provides small helpers for path generation.
//
// lazyroutes and lazycontroller use this package to support trailing URLParams
// when building named route paths. It is intentionally independent from the
// router, so packages that only need query-string appending or route-value
// splitting can use it without importing the rest of GoLazy.
package lazypath
