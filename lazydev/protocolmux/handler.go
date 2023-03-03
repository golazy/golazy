package protocolmux

import (
	"io"
	"net"
)

type listener struct {
	prefixes [][]byte
	conns    chan (*conn)
	mux      *Mux
}

func (l *listener) Accept() (net.Conn, error) {
	c, ok := <-l.conns
	if !ok {
		return nil, io.EOF
	}
	return c, nil
}

func (l *listener) Addr() net.Addr {
	return l.mux.L.Addr()
}

func (l *listener) Close() error {
	l.mux.lock.Lock()
	defer l.mux.lock.Unlock()
	for i, handler := range l.mux.listeners {
		if handler == l {
			// Remove handler from mux
			l.mux.listeners = append(l.mux.listeners[:i], l.mux.listeners[i+1:]...)
			break
		}
	}

	close(l.conns)

	return nil
}
