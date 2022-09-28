# injector

## Functions

### func [Inject](/injector.go#L35)

`func Inject(h http.Handler, header string) http.Handler`

## Types

### type [InjectHandler](/injector.go#L20)

`type InjectHandler struct { ... }`

#### func (*InjectHandler) [ServeHTTP](/injector.go#L25)

`func (ih *InjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)`

### type [Injector](/injector.go#L13)

`type Injector struct { ... }`

Injector buffers a response writer

#### func (*Injector) [Close](/injector.go#L42)

`func (i *Injector) Close()`

#### func (*Injector) [Write](/injector.go#L51)

`func (i *Injector) Write(data []byte) (int, error)`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
