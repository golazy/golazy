# devserver

package devserver implements and http and https servers with autoreload on files changes and automatic https certificate

## Functions

### func [NewDelayer](/delayer.go#L16)

`func NewDelayer(input <-chan (fsnotify.Event)) <-chan ([]fsnotify.Event)`

## Types

### type [Server](/server.go#L23)

`type Server struct { ... }`

#### func (*Server) [ListenAndServe](/server.go#L35)

`func (s *Server) ListenAndServe() error`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
