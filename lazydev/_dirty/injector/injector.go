package injector

import (
	"bytes"
	"net/http"
	"regexp"
)

var headMatch = regexp.MustCompile(`(?i)\</head\s*\>`)
var emptyHeadMatch = regexp.MustCompile(`(?i)\<head\/\>`)

// Injector buffers a response writer
type Injector struct {
	http.ResponseWriter
	AppendHeader string
	b            bytes.Buffer
	injected     bool
}

type InjectHandler struct {
	AppendHeader string
	http.Handler
}

func (ih *InjectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	injector := &Injector{
		ResponseWriter: w,
		AppendHeader:   ih.AppendHeader,
	}
	defer injector.Close()
	ih.Handler.ServeHTTP(injector, r)

}

func Inject(h http.Handler, header string) http.Handler {
	return &InjectHandler{
		Handler:      h,
		AppendHeader: header,
	}
}

func (i *Injector) Close() {
	if i.injected {
		return
	}
	if i.b.Len() != 0 {
		i.b.WriteTo(i.ResponseWriter)
	}
}

func (i *Injector) Write(data []byte) (int, error) {
	if i.injected {
		return i.ResponseWriter.Write(data)
	}

	n, err := i.b.Write(data)

	loc := headMatch.FindIndex(i.b.Bytes())
	if loc == nil {
		// Test for self closing
		loc = emptyHeadMatch.FindIndex(i.b.Bytes())
		if loc == nil {
			return n, err
		}

		i.ResponseWriter.Write(i.b.Bytes()[:loc[0]])
		i.ResponseWriter.Write([]byte("<head>"))
		i.ResponseWriter.Write([]byte(i.AppendHeader))
		i.ResponseWriter.Write([]byte("</head>"))
		i.ResponseWriter.Write(i.b.Bytes()[loc[1]:])
	} else {
		i.ResponseWriter.Write(i.b.Bytes()[:loc[0]])
		i.ResponseWriter.Write([]byte(i.AppendHeader))
		i.ResponseWriter.Write(i.b.Bytes()[loc[0]:])
	}

	i.injected = true

	return n, err
}
