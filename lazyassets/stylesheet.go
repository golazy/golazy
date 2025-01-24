package lazyassets

//
//import (
//	"bytes"
//	"io"
//	"strings"
//)
//
//type Stylesheet struct {
//	assets  *Assets
//	content [][]byte
//	Path    string
//}
//
//func (s *Stylesheet) Add(content string) *Stylesheet {
//	// TODO: Add log info about the caller
//	s.content = append(s.content, []byte(content+"\n"))
//	return s
//}
//
//func (s *Stylesheet) newReader() io.Reader {
//	data := string(bytes.Join(s.content, nil))
//	withPerma := FormatCSSPermalink(s.assets, data)
//
//	return strings.NewReader(withPerma)
//}
//
