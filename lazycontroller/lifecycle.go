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
//
// lazyroutes also recognizes controller methods named BeforeAction with
// generated arguments, such as BeforeAction(user *User) error. This interface
// remains useful for code that wants to type-check the no-argument form.
type BeforeAction interface {
	BeforeAction() error
}
