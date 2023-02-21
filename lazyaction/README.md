# lazyaction

## Variables

```golang
var Actions = lazysupport.NewStrings("Index", "Show", "Create", "Update", "Destroy", "New", "Edit")
```

```golang
var Store = sessions.NewCookieStore([]byte("//TODO: make this random and persistant"))
```

## Functions

### func [IsRouterMethod](/resource.go#L117)

`func IsRouterMethod(method string) bool`

### func [NewRouteTable](/route_tree.go#L75)

`func NewRouteTable[T any]() *routeTree[T]`

## Types

### type [Action](/action.go#L74)

`type Action struct { ... }`

#### func [NewAction](/action.go#L12)

`func NewAction(method string, r *Resource) *Action`

#### func (*Action) [NewContext](/action.go#L131)

`func (a *Action) NewContext(w http.ResponseWriter, r *http.Request) (*Context, error)`

#### func (*Action) [ServeHTTP](/action.go#L141)

`func (a *Action) ServeHTTP(w http.ResponseWriter, r *http.Request)`

#### func (*Action) [String](/action.go#L90)

`func (a *Action) String() string`

### type [Context](/context.go#L8)

`type Context struct { ... }`

#### func (*Context) [Redirect](/context.go#L17)

`func (c *Context) Redirect(url string, status int)`

### type [MemStore](/rest_controller.go#L8)

`type MemStore[T any] map[string]*T`

#### func (MemStore[T]) [Destroy](/rest_controller.go#L30)

`func (m MemStore[T]) Destroy(key string) error`

#### func (MemStore[T]) [List](/rest_controller.go#L18)

`func (m MemStore[T]) List(params ...any) ([]*T, error)`

#### func (MemStore[T]) [New](/rest_controller.go#L14)

`func (m MemStore[T]) New() *T`

#### func (MemStore[T]) [Read](/rest_controller.go#L26)

`func (m MemStore[T]) Read(key string) (*T, error)`

#### func (MemStore[T]) [Write](/rest_controller.go#L34)

`func (m MemStore[T]) Write(key string, v *T) error`

### type [Request](/request.go#L7)

`type Request struct { ... }`

#### func (*Request) [GetParam](/request.go#L11)

`func (r *Request) GetParam(name string) string`

### type [Resource](/resource.go#L11)

`type Resource struct { ... }`

#### func [NewResource](/resource.go#L28)

`func NewResource(rd *Resource) *Resource`

#### func (*Resource) [Routes](/resource.go#L207)

`func (r *Resource) Routes() string`

### type [Resource](/resource.go#L17)

`type Resource struct { ... }`

### type [ResponseWriter](/response_writer.go#L5)

`type ResponseWriter struct { ... }`

#### func (ResponseWriter) [WriteString](/response_writer.go#L9)

`func (w ResponseWriter) WriteString(s string) (int, error)`

### type [RestController](/rest_controller.go#L47)

`type RestController[T any, J Storage[T]] struct { ... }`

#### func (*RestController[T, J]) [Create](/rest_controller.go#L69)

`func (rc *RestController[T, J]) Create()`

#### func (*RestController[T, J]) [Destroy](/rest_controller.go#L76)

`func (rc *RestController[T, J]) Destroy()`

#### func (*RestController[T, J]) [Edit](/rest_controller.go#L83)

`func (rc *RestController[T, J]) Edit()`

#### func (*RestController[T, J]) [Index](/rest_controller.go#L51)

`func (rc *RestController[T, J]) Index(w http.ResponseWriter, r *http.Request) error`

#### func (*RestController[T, J]) [New](/rest_controller.go#L80)

`func (rc *RestController[T, J]) New()`

#### func (*RestController[T, J]) [Show](/rest_controller.go#L60)

`func (rc *RestController[T, J]) Show(w http.ResponseWriter, id string) error`

#### func (*RestController[T, J]) [Update](/rest_controller.go#L72)

`func (rc *RestController[T, J]) Update()`

### type [RouteDefinition](/router.go#L9)

`type RouteDefinition struct { ... }`

#### func (*RouteDefinition) [String](/router.go#L22)

`func (rd *RouteDefinition) String() string`

### type [Router](/router.go#L26)

`type Router struct { ... }`

#### func [NewRouter](/router.go#L43)

`func NewRouter() *Router`

#### func (*Router) [Add](/router.go#L54)

`func (r *Router) Add(verb, path, name, destination string, h http.Handler)`

#### func (*Router) [AddResource](/router.go#L78)

`func (router *Router) AddResource(resource *Resource)`

#### func (*Router) [AddResource](/router.go#L74)

`func (router *Router) AddResource(r *Resource)`

#### func (*Router) [LinkTo](/router.go#L116)

`func (r *Router) LinkTo(name string, params ...any) string`

LinkTo(user1, "posts", "new") => "/users/1/posts/new"
LinkTo(post1, comment1, "edit") => "/posts/1/comments/1/edit"
LinkTo("posts", "new") => "/posts/new"
LinkTo("posts", "publish") => "/posts/publish"

#### func (*Router) [ServeHTTP](/router.go#L84)

`func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request)`

### type [Session](/session.go#L10)

`type Session struct { ... }`

#### func (*Session) [AddFlash](/session.go#L60)

`func (s *Session) AddFlash(val any, vars ...string)`

#### func (*Session) [Flashes](/session.go#L54)

`func (s *Session) Flashes(vars ...string) []interface{ ... }`

#### func (*Session) [Get](/session.go#L27)

`func (s *Session) Get(key string) any`

#### func (*Session) [GetError](/session.go#L85)

`func (s *Session) GetError() string`

#### func (*Session) [GetFlash](/session.go#L70)

`func (s *Session) GetFlash(key string) string`

#### func (*Session) [GetNotice](/session.go#L93)

`func (s *Session) GetNotice() string`

#### func (*Session) [GetString](/session.go#L32)

`func (s *Session) GetString(key string) string`

#### func (*Session) [Set](/session.go#L49)

`func (s *Session) Set(key string, val any)`

#### func (*Session) [SetError](/session.go#L81)

`func (s *Session) SetError(err error)`

#### func (*Session) [SetFlash](/session.go#L65)

`func (s *Session) SetFlash(key string, val string)`

#### func (*Session) [SetNotice](/session.go#L89)

`func (s *Session) SetNotice(notice string)`

#### func (*Session) [SetString](/session.go#L45)

`func (s *Session) SetString(key, val string)`

### type [Storage](/rest_controller.go#L39)

`type Storage[K any] interface { ... }`

#### func [NewMemStore](/rest_controller.go#L10)

`func NewMemStore[T any]() Storage[T]`

### type [UrlExtractor](/action.go#L200)

`type UrlExtractor string`

#### func (UrlExtractor) [Extract](/action.go#L202)

`func (u UrlExtractor) Extract(stringArg int, paramsPosition []int) string`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
