package middlewares

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"

	"golazy.dev/lazysupport/rrrecorder"

	"github.com/timewasted/go-accept-headers"
)

const NicePanicsMiddleware = "nice_panics"

var NicePanics = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// Does the recorder supports this request?
	if !rrrecorder.IsRecordable(r) {
		return
	}

	// Does the client accept HTML?
	t, _ := accept.Negotiate(r.Header.Get("Accept"), "text/html")
	if t == "" {
		//dispatcher.Next(w, r)
		return
	}

	rec := httptest.NewRecorder()

	defer func() {
		if err := recover(); err != nil {
			w.Header().Del("Content-Length")
			w.WriteHeader(http.StatusInternalServerError)
			s := StackDecode(debug.Stack())

			p := Panic{
				Reason:     err,
				Stacktrace: s,
			}

			fmt.Println(p.String())
			w.Write([]byte(p.String()))

		} else {
			for k, v := range rec.Header() {
				for _, v := range v {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(rec.Code)
			io.Copy(w, rec.Body)
		}
	}()

	// Create new recorder
	//	dispatcher.Next(rec, r)
})

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
			sl.Func = sl.Func
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

const (
	maxLines = 5
)

func init() {
	path, _ = os.Getwd()
}

var path = ""

func (p Panic) String() string {
	s := fmt.Sprintf("panic: %s\n", p.Reason)
	if len(p.Stacktrace) > maxLines {
		p.Stacktrace = p.Stacktrace[:maxLines]
	}

	for _, sl := range p.Stacktrace {
		file := sl.File
		file, _ = filepath.Rel(path, file)

		s += fmt.Sprintf("%-55s %s\n", file+":"+sl.Line, sl.Func)
	}
	return s
}

func (sl StackLine) String() string {
	return fmt.Sprintf("%3d: %40q %45q\t%s:%s\t", sl.L, sl.Package, sl.Func, sl.File, sl.Line)
}
