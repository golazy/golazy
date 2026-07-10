// Package actioncall compiles and invokes controller action and hook call
// plans for lazyroutes.
//
// It is public so the routing package can keep the reflection-heavy planner in
// a focused subpackage, but application code should normally use lazyroutes
// instead. The supported controller action and hook contract is documented by
// lazyroutes: standard actions receive http.ResponseWriter and *http.Request,
// while generator-backed calls resolve route parameters and GenX methods.
package actioncall
