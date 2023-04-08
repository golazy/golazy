package lazyapp

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/felixge/httpsnoop"
	"golang.org/x/exp/slog"
)

// LogReqInfo describes info about HTTP request
type HTTPReqInfo struct {
	// GET etc.
	method string
	uri    string
	// response code, like 200, 404
	code int
	// number of bytes of the response sent
	size int64
	// how long did it take to
	duration time.Duration
}

func loggerMiddleware(next http.Handler) http.Handler {

	slog.SetDefault(
		slog.New(
			slog.HandlerOptions{
				AddSource: true,
			}.NewTextHandler(os.Stdout),
		),
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, r)
			return
		}
		ri := &HTTPReqInfo{
			method: r.Method,
			uri:    r.URL.String(),
		}

		m := httpsnoop.CaptureMetrics(next, w, r)

		ri.code = m.Code
		ri.size = m.Written
		ri.duration = m.Duration

		// gather information about request and log it
		slog.Info("HTTP request",
			"method", ri.method,
			"uri", ri.uri,
			"code", ri.code,
			"size", strconv.FormatInt(ri.size, 10),
			"duration", ri.duration.String())

	})

}
