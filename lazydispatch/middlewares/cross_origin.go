package middlewares

import (
	"net/http"

	"golazy.dev/lazydispatch"
)

func CrossOriginProtection(configure ...func(*http.CrossOriginProtection)) lazydispatch.Middleware {
	protection := http.NewCrossOriginProtection()
	for _, fn := range configure {
		if fn != nil {
			fn(protection)
		}
	}
	return lazydispatch.MiddlewareFunc(func(next http.Handler) http.Handler {
		return protection.Handler(next)
	})
}
