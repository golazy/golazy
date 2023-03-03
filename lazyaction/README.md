# lazyaction

## Variables

```golang
var (
    ErrNotFound      = errors.New("not found")
    ErrNotAuthorized = errors.New("not authorized")
)
```

```golang
var Actions = lazysupport.NewStringSet("Index", "Show", "Create", "Update", "Destroy", "New", "Edit")
```

```golang
var Store = sessions.NewCookieStore([]byte("//TODO: make this random and persistant"))
```

## Types

### type [Action](/action.go#L11)

`type Action struct { ... }`

#### func (*Action) [String](/action.go#L27)

`func (r *Action) String() string`

### type [Context](/context.go#L10)

`type Context struct { ... }`

#### func (*Context) [GetHeader](/context.go#L23)

`func (c *Context) GetHeader(h string) string`

#### func (*Context) [Redirect](/context.go#L31)

`func (c *Context) Redirect(url string, status int)`

#### func (*Context) [Render](/context.go#L45)

`func (c *Context) Render(data ...any)`

#### func (*Context) [Write](/context.go#L56)

`func (c *Context) Write(data []byte)`

#### func (*Context) [WriteString](/context.go#L52)

`func (c *Context) WriteString(data string)`

### type [Middleware](/middleware.go#L5)

`type Middleware interface { ... }`

### type [Request](/request.go#L7)

`type Request struct { ... }`

#### func (*Request) [GetParam](/request.go#L11)

`func (r *Request) GetParam(name string) string`

### type [Resource](/resource.go#L22)

`type Resource struct { ... }`

#### func (*Resource) [Actions](/resource.go#L108)

`func (r *Resource) Actions() []*Action`

func (r *Resource) addSubResources() {

```go
for _, sr := range r.SubResources {
	resource := NewResource(sr)
	for _, action := range resource.ResourceActions {
		a := *action
		segments := make([]string, len(r.Prefix))
		copy(segments, r.Prefix)
		segments = append(segments, r.ParamName)

		a.Path = "/" + strings.Join(segments, "/") + a.Path
		a.RouteName = r.Singular + "_" + a.ResourceName
		if a.ActionName != "" {
			a.RouteName = a.ActionName + "_" + a.RouteName
		}

		for i := range a.ParamsPosition {
			a.ParamsPosition[i] += len(segments)
		}

		a.ParamsPosition = append([]int{len(segments) - 1}, a.ParamsPosition...)

		r.ResourceActions = append(r.ResourceActions, &a)
	}
}
```

}

### type [ResourceOptions](/resource.go#L12)

`type ResourceOptions struct { ... }`

### type [ResponseWriter](/response_writer.go#L5)

`type ResponseWriter struct { ... }`

#### func (ResponseWriter) [WriteString](/response_writer.go#L9)

`func (w ResponseWriter) WriteString(s string) (int, error)`

### type [Router](/router.go#L22)

`type Router interface { ... }`

### type [Routes](/router.go#L13)

`type Routes struct { ... }`

#### func (*Routes) [Resource](/router.go#L106)

`func (r *Routes) Resource(target any, options ...*ResourceOptions)`

```go
Resource adds a resource to the router
```

The resource name is extracted from the struct name minus the Controller suffix.
If the struct is called Controller it will use the package name.

Given a struct called UserController, the follwing methods will generate the following routes:

- Index()   => GET    /users
- New()     => GET    /users/new
- Create()  => POST   /users
- Show()    => GET    /users/:id
- Edit()    => GET    /users/:id/edit
- Update()  => PUT    /users/:id
- Destroy() => DELETE /users/:id

Custom actions can be added to the resource by combining the verb and if it belongs to a Member.

- Popular() => GET /users/popular
- MemberComments() => GET /users/:id/comments
- PutMemberSuspend() => PUT /users/:id/suspend

Resource internally calls Route to add the routes to the router.

#### func (*Routes) [Route](/router.go#L33)

`func (r *Routes) Route(arguments ...any)`

Route adds routes to the router

```go
Router.Route("/", func()string{return "Hello"}) // If verb is omited it is assumed is a GET
Router.Route("POST", "/users", func() string{ return "User created!"})  // Verb can be added as a string
```

#### func (*Routes) [ServeHTTP](/router.go#L123)

`func (r *Routes) ServeHTTP(w http.ResponseWriter, req *http.Request)`

#### func (*Routes) [String](/router.go#L17)

`func (r *Routes) String() string`

### type [Session](/session.go#L10)

`type Session struct { ... }`

#### func (*Session) [AddFlash](/session.go#L68)

`func (s *Session) AddFlash(val any, vars ...string)`

#### func (*Session) [Delete](/session.go#L32)

`func (s *Session) Delete(key string)`

#### func (*Session) [Flashes](/session.go#L62)

`func (s *Session) Flashes(vars ...string) []any`

#### func (*Session) [Get](/session.go#L27)

`func (s *Session) Get(key string) any`

#### func (*Session) [GetError](/session.go#L94)

`func (s *Session) GetError() string`

#### func (*Session) [GetFlash](/session.go#L78)

`func (s *Session) GetFlash(key string) string`

#### func (*Session) [GetNotice](/session.go#L102)

`func (s *Session) GetNotice() string`

#### func (*Session) [GetString](/session.go#L40)

`func (s *Session) GetString(key string) string`

#### func (*Session) [Set](/session.go#L57)

`func (s *Session) Set(key string, val any)`

#### func (*Session) [SetError](/session.go#L90)

`func (s *Session) SetError(err error)`

#### func (*Session) [SetFlash](/session.go#L73)

`func (s *Session) SetFlash(key string, val string)`

#### func (*Session) [SetNotice](/session.go#L98)

`func (s *Session) SetNotice(notice string)`

#### func (*Session) [SetString](/session.go#L53)

`func (s *Session) SetString(key, val string)`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
