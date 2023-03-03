# protocolmux

protocolmux allows to peek the first bytes of a connection and decided to which handler forward the connection.

```go
l, _ := net.Listen("tcp", addr)

// Initialize the muuxer
mux := &Mux{L: l}

// Create listeners just by setting the prefix
helloListener := mux.ListenTo([][]byte{[]byte("ping")})

for {
	conn, _ := helloListener.Accept()
	conn.Write("pong")
	conn.Close();
}

// Or use one of the prefixes
go http.Serve(mux.ListenTo(HttpPrefix), nil) // Handle HTTP
go http.ServeTLS(mux.ListenTo(TLSPrefix), nil,...) // Handle HTTPS
```

## Variables

```golang
var (
    HTTPPrefix = [][]byte{
        []byte("GET"),
        []byte("HEAD"),
        []byte("POST"),
        []byte("PUT"),
        []byte("DELETE"),
        []byte("CONNECT"),
        []byte("OPTIONS"),
        []byte("TRACE"),
        []byte("PATCH"),
    }
    TLSPrefix = [][]byte{
        {22, 3, 0},
        {22, 3, 1},
        {22, 3, 2},
        {22, 3, 3},
    }
)
```

## Types

### type [Mux](/protocolmux.go#L41)

`type Mux struct { ... }`

#### func (*Mux) [Close](/protocolmux.go#L67)

`func (m *Mux) Close()`

#### func (*Mux) [Listen](/protocolmux.go#L75)

`func (m *Mux) Listen() error`

#### func (*Mux) [ListenTo](/protocolmux.go#L52)

`func (m *Mux) ListenTo(prefixes [][]byte) net.Listener`

---
Readme created from Go doc with [goreadme](https://github.com/posener/goreadme)
