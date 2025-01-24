package protocolmux

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
)

var logger = slog.With()

type Mux struct {
	L         net.Listener
	listeners []*listener
	closed    bool
	lock      sync.Mutex
}

type conn struct {
	net.Conn
	prefix []byte
}

func (m *Mux) ListenTo(prefixes [][]byte) net.Listener {
	h := &listener{
		prefixes: prefixes,
		conns:    make(chan (*conn)),
		mux:      m,
	}
	m.listeners = append(m.listeners, h)

	return h
}

// Close closes all the listeners, and the underlying listener
//
// # For a graceful shutdown, call Close() on all the listeners first and then call Close() on the muxer
//
// For example:
//
//	httpServer.Close()  // This will close the http listener
//	httpsServer.Close() // Close the https listener
//	muxer.Close()       // Close the underlying listener
//
// If you call Close() on the muxer first, its likely that the listeners will return an error
func (m *Mux) Close() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	// Close all the listeners
	for _, h := range m.listeners {
		h.Close()
	}
	// Mark as closed
	m.closed = true
	// Close the underlying listener
	return m.L.Close()
}

// Listen starts listening for connections in the underlying Listener
// and dispatches them to the correct handler
// It returns an error if the underlying listener returns an error
// It returns net.ErrClosed if the muxer is closed
func (m *Mux) Listen() error {
	logger.Info("starting listener", "addr", m.L.Addr())
	if m.L == nil {
		return fmt.Errorf("listener not defined")
	}

	for {
		logger.Debug("waiting for connection")
		conn, err := m.L.Accept()
		m.lock.Lock()
		var netErr *net.OpError
		if errors.As(err, &netErr) {
			return http.ErrServerClosed
		}
		if m.closed {
			if err == nil {
				conn.Close()
			}
			return net.ErrClosed
		}
		m.lock.Unlock()
		if err != nil {
			logger.Error("error while accepting connection: " + err.Error())
			return err
		}
		logger.Debug("got new connection", "addr", conn.RemoteAddr())
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
		logger.Debug("error while reading from connection", "err", err, "addr", c.RemoteAddr())
	}

	for _, handler := range m.listeners {
		for _, prefix := range handler.prefixes {
			if bytes.HasPrefix(buf[:n], prefix) {
				c := &conn{Conn: c, prefix: buf[:n]}
				handler.conns <- c
				return
			}
		}
	}
	logger.Debug("no handler found for connection", "addr", c.RemoteAddr())
	c.Close()
}
