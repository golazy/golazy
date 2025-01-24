package lazyml

import "io"

type writeSession struct {
	io.Writer
	n     int64
	err   error
	level int
}

func (w *writeSession) NewLine() {
	if Beautify {
		w.WriteS("\n")
	}
}

func (w *writeSession) WriteS(s string) (n int, err error) {
	return w.Write([]byte(s))
}

func (w *writeSession) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return
	}
	n, err = w.Writer.Write(data)
	w.n += int64(n)
	w.err = err
	return
}
