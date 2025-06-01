package lazysupport

import (
	"fmt"
	"regexp"
	"strings"
)

var funcReg = regexp.MustCompile(`(created by )?(.*\/[^\.]+)\.((\(\*?\w+\))?[^\(]*)(\(.*\))?$`)
var fileReg = regexp.MustCompile(`([\/\w\.\@]+):(\d+)`)

type StackLine struct {
	L       int
	Package string
	Func    string
	File    string
	Line    string
}

type Stacktrace struct {
	StackLines []StackLine
	pc         []uintptr
}

func (s Stacktrace) String() string {
	buf := ""
	for _, sl := range s.StackLines {
		buf += sl.String() + "\n"
	}
	return buf
}

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
