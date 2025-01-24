# router

## Variables

```golang
var Methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}
```

## Functions

### func [IsMethod](/methods.go#L5)

`func IsMethod(method string) int`

### func [NewRouteTable](/route_tree.go#L75)

`func NewRouteTable[T any]() *routeTree[T]`

## Types

### type [Router](/router.go#L7)

`type Router[T any] struct { ... }`

#### func [NewRouter](/router.go#L29)

`func NewRouter[T any]() *Router[T]`

#### func (*Router[T]) [Add](/router.go#L48)

`func (r *Router[T]) Add(verb, path string, thing *T)`

#### func (*Router[T]) [All](/router.go#L40)

`func (r *Router[T]) All() []*T`

#### func (*Router[T]) [Find](/router.go#L60)

`func (r *Router[T]) Find(verb, path string) *T`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
