// Package lazyerrors records caller context and backtraces on ordinary Go
// errors.
//
// Use New at the point where application code knows what operation failed:
//
//	return lazyerrors.New("load post %q: %w", postID, err)
//
// New formats the message like fmt.Errorf, captures the caller's stack, and
// prefixes Error with the first recorded function. A controller method that
// returns the error above might therefore render or log a message such as
// "posts.PostsController.Show: load post \"hello\": not found". The prefix is
// meant to answer "where was this error created?" without requiring every
// caller to repeat its own function name in the message.
//
// Wrapping works the same way as fmt.Errorf. If the format string contains one
// %w, the returned error implements Unwrap() error. If it contains multiple
// %w verbs, the returned error implements Unwrap() []error. That means
// errors.Is and errors.As keep working for the original cause while the
// lazyerrors wrapper adds the caller prefix and backtrace.
//
// The backtrace is intentionally small and application-facing. It starts at the
// caller of New, stores up to the internal frame limit, and is exposed through
// a Backtrace() []Frame method. There is no exported concrete error type; code
// that needs the trace should use errors.As with a local interface:
//
//	var traced interface {
//		Backtrace() []lazyerrors.Frame
//	}
//	if errors.As(err, &traced) {
//		frames := traced.Backtrace()
//		_ = frames
//	}
//
// Frame.String returns a compact "function file:line" string for logs, while
// the Function, File, and Line fields remain available to render richer output.
// Backtrace returns a copy, so callers may sort or trim the returned slice
// without mutating the error.
//
// In a GoLazy application, lazycontroller's error handling recognizes any error
// with Backtrace() []lazyerrors.Frame. When detail errors are enabled, the
// default error view receives the formatted error and backtrace data so it can
// show the source frames. lazyapp wires that controller error handling and the
// app error views into the HTTP stack; application code only needs to return or
// report the lazyerrors.New error from its controllers or services. Production
// error pages still hide details unless the app explicitly enables detailed
// errors.
//
// The package is also useful outside GoLazy. It depends only on the standard
// library, preserves normal wrapping semantics, and can be used in services,
// CLIs, workers, or tests that want errors with caller context and an inspectable
// stack without adopting the rest of the framework.
//
// Validator adds a small validation-error convention for form structs. It reads
// validate tags such as `validate:"presence;min:3;max:10"`, returns ordinary
// errors joined with errors.Join, and uses ValidationError leaves whose wrapped
// causes are typed errors such as PresenceErr, MinSizeErr, and MaxSizeErr. A
// form may also implement Validate() error for custom validations; Validator
// joins that returned error with the tag-derived failures. ErrorsFor and
// FieldErrorsFor extract validation leaves by field, and lazyapp registers
// matching `errors_for` and `field_errors_for` template helpers.
package lazyerrors
