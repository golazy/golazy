package protocolmux

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
