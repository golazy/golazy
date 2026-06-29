package lazydispatch

import (
	"bytes"
	"net/http"
	"sync"
)

const maxPooledResponseBufferBody = 64 << 10

var bufferedResponseWriterPool = sync.Pool{
	New: func() any {
		return &BufferedResponseWriter{}
	},
}

// ResponseBuffer delays sending a response until the downstream handler returns.
func ResponseBuffer() Middleware {
	return responseBuffer{}
}

type responseBuffer struct{}

func (responseBuffer) MiddlewareName() string {
	return "lazydispatch.ResponseBuffer"
}

func (responseBuffer) Handler(next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := w.(interface{ WasResponseSent() bool }); ok {
			next.ServeHTTP(w, r)
			return
		}
		buffer := acquireBufferedResponseWriter(w)
		defer releaseBufferedResponseWriter(buffer)
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
}

func NewBufferedResponseWriter(w http.ResponseWriter) *BufferedResponseWriter {
	buffer := &BufferedResponseWriter{}
	buffer.init(w)
	return buffer
}

func acquireBufferedResponseWriter(w http.ResponseWriter) *BufferedResponseWriter {
	buffer := bufferedResponseWriterPool.Get().(*BufferedResponseWriter)
	buffer.init(w)
	return buffer
}

func releaseBufferedResponseWriter(w *BufferedResponseWriter) {
	if w == nil {
		return
	}
	w.writer = nil
	if w.header != nil {
		clear(w.header)
	}
	w.status = 0
	w.sent = false
	if w.stream {
		w.stream = false
		return
	}
	if w.body.Cap() > maxPooledResponseBufferBody {
		return
	}
	w.body.Reset()
	bufferedResponseWriterPool.Put(w)
}

func (w *BufferedResponseWriter) init(writer http.ResponseWriter) {
	w.writer = writer
	if w.header == nil {
		w.header = make(http.Header)
	} else {
		clear(w.header)
	}
	w.body.Reset()
	w.status = 0
	w.sent = false
	w.stream = false
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
	clear(w.header)
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
	if status == 0 {
		status = http.StatusOK
	}
	for key, values := range w.header {
		w.writer.Header().Del(key)
		w.writer.Header()[key] = append([]string(nil), values...)
	}
	w.body.Reset()
	w.status = status
	w.sent = true
	w.stream = true
	w.writer.WriteHeader(status)
	return w.writer, nil
}
