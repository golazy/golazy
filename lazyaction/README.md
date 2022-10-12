# lazyaction

package lazyaction implements a http router that support url params and http verbs

It was design to use together lazyaction/controller:

But it can be used alone:

## Functions

### func [NewRouteTable](/route_table.go#L75)

`func NewRouteTable[T any]() *routingTable[T]`

## Types

### type [Controller](/controller.go#L10)

`type Controller struct { ... }`

#### func (Controller) [Routes](/controller.go#L39)

`func (c Controller) Routes() []*action`

```golang
rt := NewRouter()
rt.Resource(new(PostsController))

fmt.Println(rt.String())
```

 Output:

```
METHOD PATH                           DESTINATION
GET    /posts                         PostsController#Index
POST   /posts                         PostsController#Create
POST   /posts/create_super            PostsController#PostCreateSuper
GET    /posts/new                     PostsController#New
GET    /posts/:post_id                PostsController#Show
PUT    /posts/:post_id                PostsController#Update
PATCH  /posts/:post_id                PostsController#Update
DELETE /posts/:post_id                PostsController#Delete
PUT    /posts/:post_id/activate_later PostsController#MemberPutActivateLater
```

### type [Request](/request.go#L7)

`type Request struct { ... }`

#### func (*Request) [GetParam](/request.go#L11)

`func (r *Request) GetParam(name string) string`

### type [ResponseWriter](/response_writer.go#L5)

`type ResponseWriter struct { ... }`

### type [RouteAction](/router.go#L31)

`type RouteAction interface { ... }`

### type [Router](/router.go#L36)

`type Router struct { ... }`

#### func [NewRouter](/router.go#L40)

`func NewRouter() *Router`

#### func (*Router) [Add](/router.go#L51)

`func (r *Router) Add(method, path string, handler RouteAction)`

#### func (*Router) [Resource](/router.go#L136)

`func (r *Router) Resource(c interface{ ... })`

#### func (*Router) [ServeHTTP](/router.go#L107)

`func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request)`

#### func (*Router) [String](/router.go#L103)

`func (r *Router) String() string`

#### func (*Router) [Table](/router.go#L86)

`func (r *Router) Table() *lazysupport.Table`

## Sub Packages

* [go-http-routing-benchmark](./go-http-routing-benchmark)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
