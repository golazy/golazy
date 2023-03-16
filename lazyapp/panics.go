package lazyapp

import (
	"fmt"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
)

type panicWriter struct {
	http.ResponseWriter
	header http.Header
	status int
	body   []byte
}

func (pw *panicWriter) Write(p []byte) (n int, err error) {
	pw.body = append(pw.body, p...)
	return len(p), nil
}

func (pw *panicWriter) WriteHeader(statusCode int) {
	pw.status = statusCode
}

func (pw *panicWriter) Flush() {
	for k, v := range pw.header {
		pw.ResponseWriter.Header()[k] = v
	}
	if pw.status != 0 {
		pw.ResponseWriter.WriteHeader(pw.status)
	}
	pw.ResponseWriter.Write(pw.body)
	if flush, ok := pw.ResponseWriter.(http.Flusher); ok {
		flush.Flush()
	}
}

func panicMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" {
			h.ServeHTTP(w, r)
			return
		}

		defer func() {
			if err := recover(); err != nil {
				// remove all headers
				for k := range w.Header() {
					w.Header().Del(k)
				}

				w.WriteHeader(500)
				s := StackDecode(debug.Stack())

				p := Panic{
					Reason:     err,
					Stacktrace: s,
				}

				fmt.Fprint(w, p.String())
				w.Write([]byte("----------------\n"))
				w.Write(debug.Stack())
			}
		}()

		pw := &panicWriter{ResponseWriter: w}
		h.ServeHTTP(pw, r)
		pw.Flush()

	})
}

var funcReg = regexp.MustCompile(`(created by )?(.*\/[^\.]+)\.((\(\*?\w+\))?[^\(]*)(\(.*\))?$`)
var fileReg = regexp.MustCompile(`([\/\w\.\@]+):(\d+)`)

func StackDecode(data []byte) (sls []StackLine) {

	lines := strings.Split(string(data), "\n")[7:]
	for i := 0; i < len(lines); i += 2 {

		sl := StackLine{
			L: i,
		}
		s := funcReg.FindStringSubmatch(lines[i])
		if len(s) != 6 {
			continue
		} else {
			sl.Package = s[2]
			sl.Func = s[3]

			if i := strings.LastIndex(sl.Func, "."); i != -1 {
				sl.Func = sl.Func[:i] + " " + sl.Func[i+1:]
			}
			sl.Func = "func " + sl.Func + "(...)"
		}

		if len(lines) > i+1 {
			l := fileReg.FindStringSubmatch(lines[i+1])
			sl.Line = fmt.Sprint(l)
			if len(l) == 3 {
				sl.File = l[1]
				sl.Line = l[2]
			} else {
				sl.File = lines[i+1]
			}

		}

		sls = append(sls, sl)
	}

	return

}

type Panic struct {
	Reason     any
	Stacktrace []StackLine
}

type StackLine struct {
	L       int
	Package string
	Func    string
	File    string
	Line    string
}

func (p Panic) String() string {
	s := fmt.Sprintf("panic: %s\n", p.Reason)
	for _, sl := range p.Stacktrace {
		s += sl.String() + "\n"
	}
	return s
}

func (sl StackLine) String() string {
	return fmt.Sprintf("%3d: %40q %45q\t%s:%s\t", sl.L, sl.Package, sl.Func, sl.File, sl.Line)
}
