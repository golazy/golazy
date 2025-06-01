package head

import (
	"io"

	"github.com/golazy/golazy/lazyml/html"

	"github.com/golazy/golazy/lazysupport"

	"github.com/golazy/golazy/lazyml"
)

// Script represents a script tag
// See also: https://developer.mozilla.org/en-US/docs/Web/HTML/Element/script
type Script struct {

	// Content is the content of the script
	// If both Content and Src are set, two scripts will be rendered
	// Be aware that the content is not escaped
	Content string

	// Src is the source of the script
	// If both Content and Src are set, two scripts will be rendered
	Src string

	// Type will default to module. If you want to use old script, set it to
	// text/javascript
	// This attribute indicates the type of script represented. The value of this attribute will be one of the following:
	//
	// Attribute is not set (default), an empty string, or a JavaScript MIME type
	// Indicates that the script is a "classic script", containing JavaScript code. Authors are encouraged to omit the attribute if the script refers to JavaScript code rather than specify a MIME type. JavaScript MIME types are listed in the IANA media types specification.
	//
	// * importmap: This value indicates that the body of the element contains an
	// import map. The import map is a JSON object that developers can use to
	// control how the browser resolves module specifiers when importing JavaScript
	// modules.
	//
	// * module:  This value causes the code to be treated as a JavaScript module.
	// The processing of the script contents is deferred. The charset and defer
	// attributes have no effect. For information on using module, see our
	// JavaScript modules guide. Unlike classic scripts, module scripts require the
	// use of the CORS protocol for cross-origin fetching.
	//
	// speculationrules Experimental
	// This value indicates that the body of the element contains speculation rules. Speculation rules take the form of a JSON object that determine what resources should be prefetched or prerendered by the browser. This is part of the Speculation Rules API.
	//
	// Any other value
	// The embedded content is treated as a data block, and won't be processed by the browser. Developers must use a valid MIME type that is not a JavaScript MIME type to denote data blocks. All of the other attributes will be ignored, including the src attribute.
	Type string

	// This attribute explicitly indicates that certain operations should be
	// blocked on the fetching of the script. The operations that are to be
	// blocked must be a space-separated list of blocking tokens listed below.
	//
	//   render: The rendering of content on the screen is blocked.
	Blocking string

	// For classic scripts, if the async attribute is present, then the classic script will be fetched in parallel to parsing and evaluated as soon as it is available.
	//
	// For module scripts, if the async attribute is present then the scripts and all their dependencies will be fetched in parallel to parsing and evaluated as soon as they are available.
	//
	// Warning: This attribute must not be used if the src attribute is absent (i.e. for inline scripts) for classic scripts, in this case it would have no effect.
	//
	// This attribute allows the elimination of parser-blocking JavaScript where the browser would have to load and evaluate scripts before continuing to parse. defer has a similar effect in this case.
	//
	// If the attribute is specified with the defer attribute, the element will act as if only the async attribute is specified.
	//
	// This is a boolean attribute: the presence of a boolean attribute on an element represents the true value, and the absence of the attribute represents the false value.
	Async bool

	// Normal script elements pass minimal information to the window.onerror for scripts which do not pass the standard CORS checks. To allow error logging for sites which use a separate domain for static media, use this attribute. See CORS settings attributes for a more descriptive explanation of its valid arguments.
	CrossOrigin CrossOrigin

	// This Boolean attribute is set to indicate to a browser that the script is
	// meant to be executed after the document has been parsed, but before
	// firing DOMContentLoaded event.
	//
	// Scripts with the defer attribute will prevent the DOMContentLoaded event
	// from firing until the script has loaded and finished evaluating.
	//
	// Warning: This attribute must not be used if the src attribute is absent
	// (i.e. for inline scripts), in this case it would have no effect.
	//
	// The defer attribute has no effect on module scripts — they defer by
	// default.
	//
	// Scripts with the defer attribute will execute in the order in which they
	// appear in the document.
	//
	// This attribute allows the elimination of parser-blocking JavaScript where
	// the browser would have to load and evaluate scripts before continuing to
	// parse. async has a similar effect in this case.
	//
	// If the attribute is specified with the async attribute, the element will
	// act as if only the async attribute is specified.
	Defer bool

	// Provides a hint of the relative priority to use when fetching an external script. Allowed values:
	//
	//   * high Signals a high-priority fetch relative to other external scripts.
	//   * low Signals a low-priority fetch relative to other external scripts.
	//   * auto Default: Signals automatic determination of fetch priority relative to other external scripts.
	Priority Priority

	// This attribute contains inline metadata that a user agent can use to
	// verify that a fetched resource has been delivered without unexpected
	// manipulation. The attribute must not specified when the src attribute is
	// not specified. See Subresource Integrity.
	Integrity string

	// A cryptographic nonce (number used once) to allow scripts in a script-src
	// Content-Security-Policy. The server must generate a unique nonce value
	// each time it transmits a policy. It is critical to provide a nonce that
	// cannot be guessed as bypassing a resource's policy is otherwise trivial.
	Nonce string

	// Indicates which referrer to send when fetching the script, or resources fetched by the script:
	//  * no-referrer: The Referer header will not be sent.
	//  * no-referrer-when-downgrade: The Referer header will not be sent to origins without TLS (HTTPS).
	//  * origin: The sent referrer will be limited to the origin of the referring page: its scheme, host, and port.
	//  * origin-when-cross-origin: The referrer sent to other origins will be limited to the scheme, the host, and the port. Navigations on the same origin will still include the path.
	//  * same-origin: A referrer will be sent for same origin, but cross-origin requests will contain no referrer information.
	//  * strict-origin: Only send the origin of the document as the referrer when the protocol security level stays the same (HTTPS→HTTPS), but don't send it to a less secure destination (HTTPS→HTTP).
	//  * strict-origin-when-cross-origin (default): Send a full URL when performing a same-origin request, only send the origin when the protocol security level stays the same (HTTPS→HTTPS), and send no header to a less secure destination (HTTPS→HTTP).
	//  * unsafe-url: The referrer will include the origin and the path (but not the fragment, password, or username). This value is unsafe, because it leaks origins and paths from TLS-protected resources to insecure origins.
	Referrerpolicy ReferrerPolicy

	// This Boolean attribute is set to indicate that the script should not be
	// executed in browsers that support ES modules — in effect, this can be
	// used to serve fallback scripts to older browsers that do not support
	// modular JavaScript code.
	NoModule bool

	Data map[string]string
}

//go:generate stringer -type=Priority -trimprefix=Priority
type Priority int

const (
	PriorityAuto Priority = iota
	PriorityHigh
	PriorityLow
)

//go:generate stringer -type=ReferrerPolicy -trimprefix=ReferrerPolicy
type ReferrerPolicy int

const (
	// None dont set any priority
	ReferrerPolicyNone ReferrerPolicy = iota
	// NoReferrer The Referer header will not be sent.
	ReferrerPolicyNoReferrer
	// NoneWhenDowngrade The Referer header will not be sent to origins without TLS (HTTPS).
	ReferrerPolicyNoReferrerWhenDowngrade
	// Origin The sent referrer will be limited to the origin of the referring page: its scheme, host, and port.
	ReferrerPolicyOrigin
	// OriginWhenCrossOrigin The referrer sent to other origins will be limited to the scheme, the host, and the port. Navigations on the same origin will still include the path.
	ReferrerPolicyOriginWhenCrossOrigin
	// SameOrigin A referrer will be sent for same origin, but cross-origin requests will contain no referrer information.
	ReferrerPolicySameOrigin
	// StrictOrigin Only send the origin of the document as the referrer when the protocol security level stays the same (HTTPS→HTTPS), but don't send it to a less secure destination (HTTPS→HTTP).
	ReferrerPolicyStrictOrigin
	// StrictOriginWhenCrossOrigin Send a full URL when performing a same-origin request, only send the origin when the protocol security level stays the same (HTTPS→HTTPS), and send no header to a less secure destination (HTTPS→HTTP).
	ReferrerPolicyStrictOriginWhenCrossOrigin
	// UnsafeURL The referrer will include the origin and the path (but not the fragment, password, or username). This value is unsafe, because it leaks origins and paths from TLS
	ReferrerPolicyUnsafeURL
)

//go:generate stringer -type=CrossOrigin -trimprefix=CrossOrigin
type CrossOrigin int

const (
	CrossOriginDefault CrossOrigin = iota
	CrossOriginAnonymous
	// UseCredentials The browser will send cookies along with the request.
	CrossOriginUseCredentials
)

func (s *Script) element() io.WriterTo {

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
		return lazyml.NewContentNode(
			html.Script(options...),
			html.Script(html.Type("module"), lazyml.Raw(s.Content)),
		)
	}
	if s.Content != "" {
		options = append(options, lazyml.Raw(s.Content))
	}
	if s.Data != nil {
		for k, v := range s.Data {
			options = append(options, html.DataAttr(k, v))
		}
	}

	return html.Script(options...)
}

// Category returns CategoryScript
func (s Script) Category() Category {
	return HeadScript
}

// WriteTo writes the script to the writer
func (s *Script) WriteTo(w io.Writer) (n int64, err error) {
	e := s.element()
	if e == nil {
		return 0, nil
	}
	return e.WriteTo(w)
}
