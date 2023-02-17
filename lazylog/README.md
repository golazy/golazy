# lazylog

## Types

### type [Logger](/lazylog.go#L7)

`type Logger interface { ... }`

### type [Message](/lazylog.go#L22)

`type Message map[string]string`

#### func [NewMessage](/lazylog.go#L24)

`func NewMessage(args ...interface{ ... }) Message`

### type [Messager](/lazylog.go#L11)

`type Messager interface { ... }`

### type [TerminalLogger](/lazylog.go#L15)

`type TerminalLogger struct { ... }`

#### func (*TerminalLogger) [Log](/lazylog.go#L18)

`func (l *TerminalLogger) Log(m Message)`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
