# lazydev

## Functions

### func [Serve](/lazydev.go#L16)

`func Serve(handler http.Handler) error`

## Sub Packages

* [_dirty/commander](./_dirty/commander)

* [_dirty/devserver](./_dirty/devserver): package devserver implements and http and https servers with autoreload on files changes and automatic https certificate

* [_dirty/example](./_dirty/example)

* [_dirty/injector](./_dirty/injector)

* [_dirty/lazydev](./_dirty/lazydev): Package lazydev implements an autoreload server

* [_dirty/tcpdevserver](./_dirty/tcpdevserver)

* [_dirty/tcpdevserver/app](./_dirty/tcpdevserver/app)

* [_dirty/tcpdevserver/app/test](./_dirty/tcpdevserver/app/test)

* [_dirty/tcpdevserver/test](./_dirty/tcpdevserver/test)

* [_dirty/wsclients](./_dirty/wsclients): Package wsclients

* [autocerts](./autocerts): Package autocerts generates tls certificate suitable for the http server with a common certificate authority

* [filewatcher](./filewatcher): Package filewatcher notifies when the filesystem has change.

* [protocolmux](./protocolmux): protocolmux allows to peek the first bytes of a connection and decided to which handler forward the connection.

* [runner](./runner): Package runner run a restart a program on signals

* [test_app](./test_app)

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
