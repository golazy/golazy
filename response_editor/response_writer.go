/* package response_editor provides a buffered http response writer */
// It should respesct the following interfaces
// http.Flusher
// http.Pusher
// http.Hijacker
package response_editor

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
)

type Handler struct {
	Handler http.Handler
	// Configuration
}

type EditHandler struct {
	Handler http.Handler
	Edit    func(Response)
	//Async bool
}

func (h EditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eh := New(w).(*responseWriter)
	defer eh.Close()
	h.Handler.ServeHTTP(eh, r)

	if err := eh.Error(); err != nil {
		return
	}

	h.Edit(eh)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bufw := New(w)
	defer bufw.Close()
	h.Handler.ServeHTTP(bufw, r)
}

type responseWriter struct {
	body   []byte
	header http.Header
	W      http.ResponseWriter
	status int
	ignore bool

	newCount int
	// IgnoreFlush bool
}

type ResponseWriterCloser interface {
	http.ResponseWriter
	Close() error
}

func New(w http.ResponseWriter) ResponseWriterCloser {
	b, ok := w.(*responseWriter)
	if ok {
		b.newCount += 1
		return b
	}
	buffer := bufPool.Get().([]byte)
	return &responseWriter{
		W:      w,
		body:   buffer,
		header: make(http.Header),
	}
}

var bufPool = &sync.Pool{
	New: func() any {
		return make([]byte, 0, 2048)
	},
}

// Header returns the header map that will be sent by
// [responseWriter.WriteHeader]. The [Header] map also is the mechanism with which
// [Handler] implementations can set HTTP trailers.
//
// Changing the header map after a call to [responseWriter.WriteHeader] (or
// [responseWriter.Write]) has no effect unless the HTTP status code was of the
// 1xx class or the modified headers are trailers.
//
// There are two ways to set Trailers. The preferred way is to
// predeclare in the headers which trailers you will later
// send by setting the "Trailer" header to the names of the
// trailer keys which will come later. In this case, those
// keys of the Header map are treated as if they were
// trailers. See the example. The second way, for trailer
// keys not known to the [Handler] until after the first [responseWriter.Write],
// is to prefix the [Header] map keys with the [TrailerPrefix]
// constant value.
//
// To suppress automatic response headers (such as "Date"), set
// their value to nil.
func (w *responseWriter) Header() http.Header {
	return w.header
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If [responseWriter.WriteHeader] has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// [DetectContentType]. Additionally, if the total size of all written
// data is under a few KB and there are no Flush calls, the
// Content-Length header is added automatically.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (w *responseWriter) Write(data []byte) (int, error) {
	if w.ignore {
		panic("trying to write to a hijacked response")
	}
	w.body = append(w.body, data...)
	return len(data), nil
}

func (w *responseWriter) WriteHeader(statusCode int) {
	if w.ignore {
		panic("trying to write to a hijacked response")
	}
	w.status = statusCode
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.W
}

// Close writes the buffered response to the underlying http.ResponseWriter
// It will free any internal buffers, so no references to its Body should be kept
func (w *responseWriter) Close() error {
	if w.newCount != 0 {
		w.newCount--
		return nil
	}
	w.header.Set("Content-Length", fmt.Sprint(len(w.body)))

	// Set header
	for k, vv := range w.header {
		for _, v := range vv {
			w.W.Header().Add(k, v)
		}
	}
	// Set status
	if w.status != 0 {
		w.W.WriteHeader(w.status)
	}
	// Write body
	_, err := w.W.Write(w.body)

	// Put the buffer back to the pool
	bufPool.Put(w.body[:0])
	return err
}

func (w *responseWriter) Flush() {
	if w.ignore {
		panic("trying to flush a hijacked response")
	}
	if flusher, ok := w.W.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.ignore {
		panic("can't hijack the response")
	}
	if hijacker, ok := w.W.(http.Hijacker); ok {
		conn, rw, err := hijacker.Hijack()
		w.ignore = true
		return conn, rw, err
	}
	return nil, nil, errors.New("response writer does not support hijacking")
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Body() *[]byte {
	return &w.body
}

func (w *responseWriter) Error() error {
	return nil
}

var _ http.ResponseWriter = &responseWriter{}
var _ Response = &responseWriter{}

type Response interface {
	Error() error

	Body() *[]byte
	Header() http.Header
	Status() int
	WriteHeader(int)
}

var ErrResponseIsNotEditable = errors.New("response is not editable")
