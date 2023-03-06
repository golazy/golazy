# assets

## Types

### type [Manager](/assets.go#L29)

`type Manager struct { ... }`

#### func (*Manager) [Add](/assets.go#L83)

`func (am *Manager) Add(name, content string)`

#### func (*Manager) [AddFS](/assets.go#L95)

`func (am *Manager) AddFS(files fs.ReadDirFS, prefix ...string)`

#### func (*Manager) [AddReader](/assets.go#L87)

`func (am *Manager) AddReader(name string, r io.Reader)`

#### func (*Manager) [FullPath](/assets.go#L162)

`func (am *Manager) FullPath(name string) string`

#### func (*Manager) [Has](/assets.go#L215)

`func (am *Manager) Has(path string) bool`

#### func (*Manager) [Permalink](/assets.go#L166)

`func (am *Manager) Permalink(name string) string`

#### func (*Manager) [ServeHTTP](/assets.go#L176)

`func (am *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request)`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
