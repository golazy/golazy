package protocolmux

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

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
		[]byte{22, 3, 0},
		[]byte{22, 3, 1},
		[]byte{22, 3, 2},
		[]byte{22, 3, 3},
	}
)

type handler struct {
	prefixes [][]byte
	conns    chan (*conn)
	mux      *Mux
}

func (h *handler) Accept() (net.Conn, error) {
	c, ok := <-h.conns
	if !ok {
		return nil, io.EOF
	}
	return c, nil
}

func (h *handler) Addr() net.Addr {
	return h.mux.L.Addr()
}

func (h *handler) Close() error {
	for i, handler := range h.mux.handlers {
		if handler == h {
			h.mux.handlers = append(h.mux.handlers[:i], h.mux.handlers[i+1:]...)
		}
	}

	close(h.conns)

	return nil
}

type Mux struct {
	L        net.Listener
	handlers []*handler
}

type conn struct {
	net.Conn
	prefix []byte
}

func (m *Mux) ListenTo(prefixes [][]byte) net.Listener {
	h := &handler{
		prefixes: prefixes,
		conns:    make(chan (*conn)),
		mux:      m,
	}
	m.handlers = append(m.handlers, h)

	return h
}

func (m *Mux) l(v ...interface{}) {
	fmt.Println(append([]interface{}{"MUX:"}, v...)...)
}

func (m *Mux) Listen() error {
	m.l("starting listener")
	if m.L == nil {
		return fmt.Errorf("Listener not defined")
	}

	for {
		m.l("Waiting for connection")
		conn, err := m.L.Accept()
		m.l("Got new Connection", conn.RemoteAddr())
		fmt.Println()
		if err != nil {
			return err
		}
		go m.handleConn(conn)
	}
}

func (c *conn) Read(b []byte) (int, error) {
	if len(c.prefix) == 0 {
		return c.Conn.Read(b)
	}
	n := copy(b, c.prefix)
	c.prefix = c.prefix[n:] // BUG?
	return n, nil
}

func (m *Mux) handleConn(c net.Conn) {
	buf := make([]byte, 1024)
	n, err := c.Read(buf)
	if err != nil {
		log.Println("erro while handling a connection:", err)
	}

	for _, handler := range m.handlers {
		for _, prefix := range handler.prefixes {
			if bytes.HasPrefix(buf[:n], prefix) {
				c := &conn{Conn: c, prefix: buf[:n]}
				handler.conns <- c
				return
			}
		}
	}
	log.Println("no protocol for connection")
	c.Close()
}
