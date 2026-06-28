// Package lazylogs provides slog-compatible context logging helpers.
package lazylogs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	"golazy.dev/lazytelemetry/lazytracing"
)

type contextKey struct{}

// NewTextLogger returns a slog logger that writes text records.
func NewTextLogger(out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	return slog.New(slog.NewTextHandler(out, nil))
}

// NewJSONLogger returns a slog logger that writes JSON records.
func NewJSONLogger(out io.Writer) *slog.Logger {
	if out == nil {
		out = os.Stdout
	}
	return slog.New(slog.NewJSONHandler(out, nil))
}

// WithLogger attaches logger to ctx.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, logger)
}

// Logger returns the logger attached to ctx, or slog.Default when none exists.
func Logger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}

// WithAttrs returns a context whose logger includes attrs.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}
	return WithLogger(ctx, Logger(ctx).With(args...))
}

// WithTags returns a context whose logger includes the given tags.
func WithTags(ctx context.Context, tags ...string) context.Context {
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			clean = append(clean, tag)
		}
	}
	if len(clean) == 0 {
		return ctx
	}
	return WithAttrs(ctx, slog.Any("tags", clean))
}

// WithGroup returns a context whose logger writes subsequent attrs in group.
func WithGroup(ctx context.Context, group string) context.Context {
	group = strings.TrimSpace(group)
	if group == "" {
		return ctx
	}
	return WithLogger(ctx, Logger(ctx).WithGroup(group))
}

// LogAttrs logs with the context logger and records the message as a span event
// when ctx has an active span.
func LogAttrs(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) {
	Logger(ctx).LogAttrs(ctx, level, message, attrs...)
	if span := lazytracing.SpanFromContext(ctx); span != nil {
		eventAttrs := make([]slog.Attr, 0, len(attrs)+3)
		eventAttrs = append(eventAttrs,
			slog.String("level", strings.ToLower(level.String())),
			slog.String("message", message),
		)
		if requestID := lazytracing.RequestID(ctx); requestID != "" && !hasAttr(attrs, "request_id") {
			eventAttrs = append(eventAttrs, slog.String("request_id", requestID))
		}
		eventAttrs = append(eventAttrs, attrs...)
		span.AddEvent("log", eventAttrs...)
		lazytracing.Log(ctx, "log", strings.ToLower(level.String())+" "+message)
	}
}

// Debug logs a debug message.
func Debug(ctx context.Context, message string, attrs ...slog.Attr) {
	LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}

// Info logs an info message.
func Info(ctx context.Context, message string, attrs ...slog.Attr) {
	LogAttrs(ctx, slog.LevelInfo, message, attrs...)
}

// Warn logs a warning message.
func Warn(ctx context.Context, message string, attrs ...slog.Attr) {
	LogAttrs(ctx, slog.LevelWarn, message, attrs...)
}

// Error logs an error message.
func Error(ctx context.Context, message string, attrs ...slog.Attr) {
	LogAttrs(ctx, slog.LevelError, message, attrs...)
}

func hasAttr(attrs []slog.Attr, name string) bool {
	for _, attr := range attrs {
		if attr.Key == name {
			return true
		}
	}
	return false
}
