# router

package lazyaction implements a http router that support url params and http verbs

It was design to use together lazyaction/controller:

But it can be used alone:

```go
say := func(what string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprint(what, r.Form)))
	}
}

router := &Router{
	PrefixCatchAll{"page_id", Routes{
		RoutePath{"", "", say("show_page")},
		RoutePath{"publish", "", say("publish_page")},
	}},
	RoutePath{"posts", "", say("posts_index")},
	Prefix{"posts", Routes{
		RouteCatchAll{"post_id", "", say("post_show")},
		RoutePath{"", "", say("post index")},
		RoutePath{"new", "", say("post new")},
		PrefixCatchAll{"post_id", Routes{
			RoutePath{"publish", "", say("publish")},
		}},
		RoutePath{"publish", "", say("publish")},
	}},
}
```

## Types

### type [CatchAllPath](/router.go#L66)

`type CatchAllPath struct { ... }`

### type [CatchAllPrefix](/router.go#L78)

`type CatchAllPrefix struct { ... }`

### type [Handler](/router.go#L40)

`type Handler interface { ... }`

### type [HandlerFunc](/router.go#L44)

`type HandlerFunc func(w ResponseWriter, r *Request)`

#### func (HandlerFunc) [ServeHTTP](/router.go#L46)

`func (h HandlerFunc) ServeHTTP(w ResponseWriter, r *Request)`

### type [Method](/router.go#L50)

`type Method string`

### type [Path](/router.go#L101)

`type Path struct { ... }`

### type [PathSegment](/router.go#L62)

`type PathSegment interface{ ... }`

### type [Prefix](/router.go#L73)

`type Prefix struct { ... }`

### type [RedirectPath](/router.go#L83)

`type RedirectPath struct { ... }`

#### func (RedirectPath) [ServeHTTP](/router.go#L89)

`func (rp RedirectPath) ServeHTTP(w ResponseWriter, r *Request)`

### type [Request](/request.go#L8)

`type Request struct { ... }`

#### func (*Request) [GetParam](/request.go#L13)

`func (r *Request) GetParam(name string) string`

### type [ResponseWriter](/response_writer.go#L5)

`type ResponseWriter struct { ... }`

### type [Router](/router.go#L60)

`type Router Routes`

#### func (Router) [ServeHTTP](/router.go#L108)

`func (router Router) ServeHTTP(w http.ResponseWriter, r *http.Request)`

### type [Routes](/router.go#L64)

`type Routes []PathSegment`

```golang
say := func(what string) HandlerFunc {
    return func(w ResponseWriter, r *Request) {
        params, _ := json.Marshal(r.Params)
        w.Write([]byte(fmt.Sprintf("%s Params: %s", what, params)))
    }
}

router := &Router{
    Path{"", "GET", "", say("Home page")},
    Path{"pages", "", "", say("Pages index")}, // HTTP Medoth defaults to GET
    Prefix{"pages", Routes{ // Handles `pages/`
        RedirectPath{
            Path: "",
            To:   "../pages",
        },
        CatchAllPath{"page_id", "", "", say("Nice Page")}, // Matches `/pages/:page_id` assiging `page_id` to httpRequest.Form.Values.Get("page_id")
        CatchAllPrefix{"page_id", Routes{
            Path{"share", "POST", "", say("Page shared!")}, // Matches `POST /pages/:page_id/share`
            Prefix{"paragraphs", Routes{
                CatchAllPath{"paragraph_id", "", "", say("Paragraph")},
            }},
        }},
    }},
}

// For testing
query := func(path string) string {
    w := httptest.NewRecorder()
    r := httptest.NewRequest("GET", path, nil)
    router.ServeHTTP(w, r)
    return w.Body.String()

}

fmt.Println(query("/pages/33"))
fmt.Println(query("/pages/33/paragraph/42.json"))
```

 Output:

```
Nice Page Params: {"page_id":["33"]}
Paragraph Params: {"format":["json"],"page_id":["33"],"paragraph_id":["42"]}
```

#### func [Resource](/resource.go#L10)

`func Resource(Controller interface{ ... }) Routes`

```golang
routes := Resource(new(PostsController))
fmt.Println(routes)
```

 Output:

```
METHOD PATH                           DESTINATION
POST   /posts                         PostsController#Create
GET    /posts                         PostsController#Index
POST   /posts/create_super            PostsController#PostCreateSuper
GET    /posts/new                     PostsController#New
DELETE /posts/:post_id                PostsController#Delete
GET    /posts/:post_id                PostsController#Show
PUT    /posts/:post_id                PostsController#Update
PATCH  /posts/:post_id                PostsController#Update
PUT    /posts/:post_id/activate_later PostsController#MemberPutActivateLater
```

#### func (Routes) [String](/route_string.go#L10)

`func (r Routes) String() string`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
