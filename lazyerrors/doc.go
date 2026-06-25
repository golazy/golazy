// Package lazyerrors records caller context and backtraces for application
// errors.
//
// Use New where application code wants to return an error through the GoLazy
// controller or service stack with enough local context for debugging:
//
//	return lazyerrors.New("load post %q: %w", postID, err)
//
// The returned error message is prefixed with the caller, such as
// "posts.PostsController.Show: load post ...". Formatting follows fmt.Errorf,
// including support for %w wrapping. Errors with one wrapped cause implement
// Unwrap() error; errors with multiple wrapped causes implement Unwrap()
// []error. Code that needs the recorded trace can use errors.As with a local
// interface{ Backtrace() []lazyerrors.Frame }. Frame implements String for
// compact log output and also exposes Function, File, and Line fields.
package lazyerrors
