package lazycontroller

import (
	"net/http"

	"golazy.dev/lazyview"
)

// RequestBinder prepares a controller instance for one request.
type RequestBinder interface {
	BindRequest(http.ResponseWriter, *http.Request, lazyview.Route) error
}

// RequestResetter clears request-specific references before a controller returns to a pool.
type RequestResetter interface {
	ResetRequest()
}

// BeforeAction runs after request binding and before the routed action.
type BeforeAction interface {
	BeforeAction() error
}
