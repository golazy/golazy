# lazydev

```go
Package lazydev implements an autoreload server
```

The main method Serve() will start a child process with `go run *.go` (except
test files). everytime there is a change in the filesystem, it will stop the
server and start again.

The default port can be changed through the PORT environment variable or through DefaultListenAddr

```go
lazydev.DefaultListenAddr = ":9090"
```

By default it uses http.DefaultServeMux but can be changed through DefaultServerMux

```go
lazydev.DefaultServeMux = http.HandlerFunc(func(w http.RespnoseWriter, r *http.Request){w.Write([]byte("hello"))})
```

It watches for changes in WatchPaths that defaults to the current directory. More can be added by modifing the variable or by the LAZYWATCH environment variable.

```go
LAZYWATCH=""./..." go run *.go
```

## Functions

### func [Serve](/serve.go#L22)

`func Serve(h http.Handler)`

## Sub Packages

* [devserver](./devserver)

* [devserver/autocerts](./devserver/autocerts)

* [devserver/protocolmux](./devserver/protocolmux)

* [devserver/tcpdevserver](./devserver/tcpdevserver)

* [devserver/tcpdevserver/app](./devserver/tcpdevserver/app)

* [devserver/tcpdevserver/app/test](./devserver/tcpdevserver/app/test)

* [devserver/tcpdevserver/test](./devserver/tcpdevserver/test)

* [injector](./injector)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
