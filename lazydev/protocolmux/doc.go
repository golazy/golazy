// protocolmux allows to peek the first bytes of a connection and decided to which handler forward the connection.
/*

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


*/
package protocolmux
