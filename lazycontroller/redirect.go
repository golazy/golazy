package lazycontroller

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type redirectOptions struct {
	status []int
}

// RedirectOption customizes route redirect helpers.
type RedirectOption interface {
	applyRedirectOption(*redirectOptions)
}

type redirectStatusOption int

func (option redirectStatusOption) applyRedirectOption(options *redirectOptions) {
	options.status = append(options.status, int(option))
}

// RedirectStatus sets the status used by RedirectToRoute.
func RedirectStatus(code int) RedirectOption {
	return redirectStatusOption(code)
}

// Redirect sends an HTTP redirect and marks the controller response as written.
// It defaults to 302 Found. Pass a single 3xx status such as
// http.StatusMovedPermanently or http.StatusSeeOther to override it.
func (b *Base) Redirect(location string, status ...int) error {
	if b.writer == nil || b.request == nil {
		return fmt.Errorf("controller base is not initialized")
	}
	code, err := redirectStatus(status...)
	if err != nil {
		return err
	}
	if err := validateRedirectLocation(location); err != nil {
		return err
	}
	http.Redirect(b.writer, b.request, location, code)
	return nil
}

// RedirectTo is an alias for Redirect for controllers that prefer Rails-style
// wording in action code.
func (b *Base) RedirectTo(location string, status ...int) error {
	return b.Redirect(location, status...)
}

// RedirectToRoute redirects to a named route built with PathFor.
func (b *Base) RedirectToRoute(name string, values ...any) error {
	routeValues, options := splitRedirectValues(values)
	path, err := b.PathFor(name, routeValues...)
	if err != nil {
		return err
	}
	return b.RedirectTo(path, options.status...)
}

// PermanentRedirectTo redirects permanently to location with 301 Moved Permanently.
func (b *Base) PermanentRedirectTo(location string) error {
	return b.RedirectTo(location, http.StatusMovedPermanently)
}

// PermanentRedirectToRoute permanently redirects to a named route built with PathFor.
func (b *Base) PermanentRedirectToRoute(name string, values ...any) error {
	routeValues, options := splitRedirectValues(values)
	if len(options.status) > 0 {
		return fmt.Errorf("permanent route redirects do not accept RedirectStatus")
	}
	path, err := b.PathFor(name, routeValues...)
	if err != nil {
		return err
	}
	return b.PermanentRedirectTo(path)
}

// RedirectBackOrTo redirects to the same-host Referer header when present,
// otherwise it redirects to fallbackLocation.
func (b *Base) RedirectBackOrTo(fallbackLocation string, status ...int) error {
	location := ""
	if b.request != nil {
		location = b.URLFrom(b.request.Referer())
	}
	if location == "" {
		location = fallbackLocation
	}
	return b.Redirect(location, status...)
}

// RedirectBack is an alias for RedirectBackOrTo.
func (b *Base) RedirectBack(fallbackLocation string, status ...int) error {
	return b.RedirectBackOrTo(fallbackLocation, status...)
}

// URLFrom returns location when it is safe to use as an internal redirect
// target for the current request. Absolute URLs must match the request host;
// relative URLs must be absolute paths such as "/posts".
func (b *Base) URLFrom(location string) string {
	if b.request == nil {
		return ""
	}
	if err := validateRedirectLocation(location); err != nil {
		return ""
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return ""
	}
	if parsed.Host != "" {
		if sameHost(parsed.Host, b.request.Host) {
			return location
		}
		return ""
	}
	if parsed.Scheme != "" {
		return ""
	}
	if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, "//") {
		return location
	}
	return ""
}

func redirectStatus(status ...int) (int, error) {
	switch len(status) {
	case 0:
		return http.StatusFound, nil
	case 1:
		code := status[0]
		if code < 300 || code >= 400 || code == http.StatusNotModified {
			return 0, fmt.Errorf("redirect status must be a 3xx redirect status, got %d", code)
		}
		return code, nil
	default:
		return 0, fmt.Errorf("redirect accepts at most one status")
	}
}

func splitRedirectValues(values []any) ([]any, redirectOptions) {
	var options redirectOptions
	routeValues := make([]any, 0, len(values))
	for _, value := range values {
		if option, ok := value.(RedirectOption); ok {
			option.applyRedirectOption(&options)
			continue
		}
		routeValues = append(routeValues, value)
	}
	return routeValues, options
}

func validateRedirectLocation(location string) error {
	if strings.TrimSpace(location) == "" {
		return fmt.Errorf("redirect location is required")
	}
	for _, r := range location {
		if r < 0x20 || r == 0x7f {
			return fmt.Errorf("redirect location contains an unsafe header character")
		}
	}
	parsed, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("redirect location is invalid: %w", err)
	}
	if parsed.Scheme == "" && parsed.Host == "" && !strings.HasPrefix(location, "/") {
		return fmt.Errorf("redirect location must be an absolute URL or absolute path")
	}
	if parsed.Scheme != "" && parsed.Host == "" {
		return fmt.Errorf("redirect location must include a host")
	}
	return nil
}

func sameHost(a string, b string) bool {
	return strings.EqualFold(a, b)
}
