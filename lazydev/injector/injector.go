package injector

import (
	"io"
	"net/http"
	"unicode"
)

func New(content io.WriterTo) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ri := &ResponseInjector{
				ResponseWriter: w,
				content:        content,
			}
			h.ServeHTTP(ri, r)
		})
	}
}

type ResponseInjector struct {
	pos int
	http.ResponseWriter
	content io.WriterTo
	inuse   bool
	done    bool
}

func (ri *ResponseInjector) reset(h http.ResponseWriter) {
	ri.ResponseWriter = h
	ri.pos = 0
	ri.inuse = false
	ri.done = false
}

var head = "<body"

func (ri *ResponseInjector) Flush() {
	if !ri.done {
		ri.ResponseWriter.Write([]byte(head[:ri.pos]))
	}
	ri.done = true
	if f, ok := ri.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (ri *ResponseInjector) Write(data []byte) (int, error) {
	if ri.done {
		return ri.ResponseWriter.Write(data)
	}
	if !ri.inuse {
		ri.ResponseWriter.Header().Del("Content-Length")

		ct := ri.ResponseWriter.Header().Get("Content-Type")
		if ct != "" && ct != "text/html" && ct != "application/xhtml+xml" {
			ri.done = true
		}
		ri.inuse = true
	}

	max := len(head) - 1

	for i, r := range string(data) {
		currentR := rune(head[ri.pos])
		if currentR == unicode.ToLower(r) {
			if ri.pos == max {
				ri.content.WriteTo(ri.ResponseWriter)
				ri.ResponseWriter.Write([]byte(head))
				ri.ResponseWriter.Write(data[i+1:])
				ri.done = true
				return len(data), nil
			} else {
				ri.pos++
			}
		} else {
			ri.ResponseWriter.Write([]byte(head[:ri.pos]))
			ri.ResponseWriter.Write([]byte(string([]rune{r})))
			ri.pos = 0
		}

	}

	return len(data), nil
}
