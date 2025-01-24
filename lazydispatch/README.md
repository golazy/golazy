# Lazy Dispatch

lazydispatch is a module of golazy to route http request into user defined actions.

This document covers the funcionality of lazydispatch alone. Please read the golazy intro first before aproaching this document


* Intro
* Creating Routes
* Using controllers
* The base controller
* The controller model
* Testing actions
* Using it without the rest of golazy

# Intro


```
  d := lazydispatcher.New()
  d.Draw(func (drawer *lazydispatcher.Scope){
    drawer.Resources(&PostsController{})
  })
```

And a controller defined as:

```
type PostsController struct {}

func (c *PostsController) Index(){}

func (c *PostsController)



# base.Controller




* base.Controller to handle all things related with the request.




# lazydispatch.Dispatcher



```go

func main() {
	dispatcher := lazydispatch.New()
	dispatcher.Use(middlewares.Logger)
	dispatcher.Draw(func( routes lazydispatch.Scope){
		routes.Resources(&PostsController{})
	})
	http.ListenAndServe(":2000", dispatcher)
}

var postsDB = map[string]string{
	{"hello", "hi this is my first post"},
	{"golazy", "the fastests way to do webpages" },
}

type PostsController struct {

}

func (c *PostsController) Index() {

}

func (c *PostsController) Show(id string) error {



}

```


*/