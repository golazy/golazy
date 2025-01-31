# protocolmux

## Description

protocolmux is a package that allows you to peek the first bytes of a connection and decide to which handler to forward the connection. It is useful for handling multiple protocols on the same port.

## Usage

### Example

Here's an example of how to use protocolmux:

```go
package main

import (
	"net"
	"net/http"
	"golazy.dev/protocolmux"
)

func main() {
	l, _ := net.Listen("tcp", ":8080")

	// Initialize the muxer
	mux := &protocolmux.Mux{L: l}

	// Create listeners just by setting the prefix
	helloListener := mux.ListenTo([][]byte{[]byte("ping")})

	go func() {
		for {
			conn, _ := helloListener.Accept()
			conn.Write([]byte("pong"))
			conn.Close()
		}
	}()

	// Or use one of the predefined prefixes
	go http.Serve(mux.ListenTo(protocolmux.HTTPPrefix), nil) // Handle HTTP
	go http.ServeTLS(mux.ListenTo(protocolmux.TLSPrefix), nil, "cert.pem", "key.pem") // Handle HTTPS

	mux.Listen()
}
```

## Dependencies and Installation

To use protocolmux, you need to have Go installed on your system. You can install protocolmux using the following command:

```sh
go get golazy.dev/protocolmux
```

## Contributing and Reporting Issues

If you would like to contribute to the development of protocolmux or report any issues, please visit the [GitHub repository](https://github.com/golazy/golazy) and follow the guidelines for contributing and reporting issues.
