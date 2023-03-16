package script

import (
	"fmt"
	"io"
	"net/url"

	"golazy.dev/lazysupport"
	"golazy.dev/lazyview/html"
	"golazy.dev/lazyview/nodes"
)

type Asset string

type Script struct {

	// Content is the content of the script
	Content string
	// Src is the source of the script
	// If both Content and Src are set, two scripts will be rendered
	Src string
	// Type will default to module. If you want to use old script, set it to text/javascript
	Type           string
	Blocking       string
	Async          bool
	CrossOrigin    CrossOrigin
	Defer          bool
	Priority       Priority
	Integrity      string
	Nonce          string
	Referrerpolicy ReferrerPolicy
	NoModule       bool
}

//go:generate stringer -type=Priority
type Priority int

const (
	Auto Priority = iota
	High
	Low
)

//go:generate stringer -type=ReferrerPolicy
type ReferrerPolicy int

const (
	// None dont set any priority
	None ReferrerPolicy = iota
	// NoReferrer The Referer header will not be sent.
	NoReferrer
	// NoneWhenDowngrade The Referer header will not be sent to origins without TLS (HTTPS).
	NoReferrerWhenDowngrade
	// Origin The sent referrer will be limited to the origin of the referring page: its scheme, host, and port.
	Origin
	// OriginWhenCrossOrigin The referrer sent to other origins will be limited to the scheme, the host, and the port. Navigations on the same origin will still include the path.
	OriginWhenCrossOrigin
	// SameOrigin A referrer will be sent for same origin, but cross-origin requests will contain no referrer information.
	SameOrigin
	// StrictOrigin Only send the origin of the document as the referrer when the protocol security level stays the same (HTTPS→HTTPS), but don't send it to a less secure destination (HTTPS→HTTP).
	StrictOrigin
	// StrictOriginWhenCrossOrigin Send a full URL when performing a same-origin request, only send the origin when the protocol security level stays the same (HTTPS→HTTPS), and send no header to a less secure destination (HTTPS→HTTP).
	StrictOriginWhenCrossOrigin
	// UnsafeURL The referrer will include the origin and the path (but not the fragment, password, or username). This value is unsafe, because it leaks origins and paths from TLS
	UnsafeURL
)

//go:generate stringer -type=CrossOrigin
type CrossOrigin int

const (
	DefaultCrossOrigin CrossOrigin = iota
	Anonymous
	// UseCredentials The browser will send cookies along with the request.
	UseCredentials
)

func (s *Script) Element() io.WriterTo {

	options := []any{}
	if s.Src == "" && s.Content == "" {
		return nil
	}

	if s.Src != "" {
		options = append(options, html.Src(s.Src))
	}

	if s.Type == "" {
		if !s.NoModule {
			options = append(options, html.Type("module"))
		}
	} else {
		options = append(options, html.Type(s.Type))
	}

	if s.Blocking != "" {
		options = append(options, html.Blocking(s.Blocking))
	}

	if s.Async {
		options = append(options, html.Async())
	}

	if s.CrossOrigin != 0 {
		options = append(options, html.Crossorigin(lazysupport.Dasherize(s.CrossOrigin.String())))
	}

	if s.Defer {
		options = append(options, html.Defer())
	}

	if s.Integrity != "" {
		options = append(options, html.Integrity(s.Integrity))
	}
	if s.Nonce != "" {
		options = append(options, html.Nonce(s.Nonce))
	}
	if s.Referrerpolicy != 0 {
		rp := lazysupport.Dasherize(s.Referrerpolicy.String())
		options = append(options, html.Referrerpolicy(rp))
	}

	if s.NoModule {
		options = append(options, html.Nomodule())
	}

	if s.Src != "" && s.Content != "" {
		return nodes.NewContentNode(
			html.Script(options...),
			html.Script(html.Type("module"), nodes.Raw(s.Content)),
		)
	}
	if s.Content != "" {
		options = append(options, nodes.Raw(s.Content))
	}

	return html.Script(options...)
}

func (s *Script) WriteTo(w io.Writer) (n int64, err error) {
	e := s.Element()
	if e == nil {
		return 0, nil
	}
	return s.Element().WriteTo(w)
}

func New(arg any) Script {
	switch arg := arg.(type) {
	case *Script:
		return *arg
	case Script:
		return arg
	case []byte:
		return Script{
			Content: string(arg),
		}
	case string:
		u, err := url.Parse(arg)
		if err != nil {
			return Script{
				Content: arg,
			}
		}
		return Script{
			Src: u.String(),
		}
	default:
		panic(fmt.Sprintf("script.New: invalid argument type %T", arg))
	}

}
