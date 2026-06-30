package lazydispatch

import (
	"bytes"
	"net/http"
)

const defaultResponseBufferLimit = 1 << 20

// ResponseBuffer delays sending a response until the downstream handler returns.
// Responses larger than the in-memory limit are streamed to the client to avoid
// unbounded memory growth.
func ResponseBuffer() Middleware {
	return responseBuffer{}
}

type responseBuffer struct{}

func (responseBuffer) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(interface{ WasResponseSent() bool }); ok {
			next.ServeHTTP(w, r)
			return
		}
		buffer := NewBufferedResponseWriter(w)
		next.ServeHTTP(buffer, r)
		_ = buffer.Flush()
	})
}

// BufferedResponseWriter is a response writer whose headers and body can be reset.
type BufferedResponseWriter struct {
	writer http.ResponseWriter
	header http.Header
	body   bytes.Buffer
	status int
	sent   bool
	stream bool
	limit  int
}

func NewBufferedResponseWriter(w http.ResponseWriter) *BufferedResponseWriter {
	return &BufferedResponseWriter{
		writer: w,
		header: make(http.Header),
		limit:  defaultResponseBufferLimit,
	}
}

func (w *BufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *BufferedResponseWriter) Write(data []byte) (int, error) {
	if w.stream {
		return len(data), nil
	}
	if !w.sent {
		w.WriteHeader(http.StatusOK)
	}
	if w.limit > 0 && w.body.Len()+len(data) > w.limit {
		stream, err := w.startStream(w.status, true)
		if err != nil {
			return 0, err
		}
		return stream.Write(data)
	}
	return w.body.Write(data)
}

func (w *BufferedResponseWriter) WriteHeader(status int) {
	if w.sent {
		return
	}
	w.status = status
	w.sent = true
}

func (w *BufferedResponseWriter) WasResponseSent() bool {
	return w.sent
}

func (w *BufferedResponseWriter) Reset() {
	if w.stream {
		return
	}
	w.header = make(http.Header)
	w.body.Reset()
	w.status = 0
	w.sent = false
}

func (w *BufferedResponseWriter) Flush() error {
	if w.stream {
		return nil
	}
	if !w.sent && len(w.header) == 0 {
		return nil
	}
	for key, values := range w.header {
		w.writer.Header().Del(key)
		w.writer.Header()[key] = append([]string(nil), values...)
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.writer.WriteHeader(w.status)
	if w.body.Len() == 0 {
		return nil
	}
	_, err := w.writer.Write(w.body.Bytes())
	return err
}

func (w *BufferedResponseWriter) Unwrap() http.ResponseWriter {
	return w.writer
}

// StartStream commits the buffered headers and lets callers write directly to
// the wrapped response writer.
func (w *BufferedResponseWriter) StartStream(status int) (http.ResponseWriter, error) {
	return w.startStream(status, false)
}

func (w *BufferedResponseWriter) startStream(status int, flushBuffered bool) (http.ResponseWriter, error) {
	if status == 0 {
		status = http.StatusOK
	}
	for key, values := range w.header {
		w.writer.Header().Del(key)
		w.writer.Header()[key] = append([]string(nil), values...)
	}
	var buffered []byte
	if flushBuffered {
		buffered = append([]byte(nil), w.body.Bytes()...)
	}
	w.body.Reset()
	w.status = status
	w.sent = true
	w.stream = true
	w.writer.WriteHeader(status)
	if len(buffered) == 0 {
		return w.writer, nil
	}
	_, err := w.writer.Write(buffered)
	return w.writer, err
}
