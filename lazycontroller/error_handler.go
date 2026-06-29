package lazycontroller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
)

type requestErrorStateKey struct{}

type requestErrorState struct {
	controller any
	err        error
}

var requestErrorStatePool = sync.Pool{
	New: func() any {
		return &requestErrorState{}
	},
}

type controllerErrorHandler interface {
	HandleError(http.ResponseWriter, *http.Request, error) error
}

func ErrorHandler(ctx context.Context) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			state := acquireRequestErrorState()
			r = r.WithContext(context.WithValue(r.Context(), requestErrorStateKey{}, state))

			defer func() {
				defer releaseRequestErrorState(state)
				if recovered := recover(); recovered != nil {
					state.err = PanicError(recovered)
				}
				if state.err == nil {
					return
				}
				handleReportedError(ctx, w, r, state.controller, state.err)
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func acquireRequestErrorState() *requestErrorState {
	return requestErrorStatePool.Get().(*requestErrorState)
}

func releaseRequestErrorState(state *requestErrorState) {
	if state == nil {
		return
	}
	state.controller = nil
	state.err = nil
	requestErrorStatePool.Put(state)
}

func ReportController(r *http.Request, controller any) bool {
	state, ok := errorStateFromRequest(r)
	if !ok {
		return false
	}
	state.controller = controller
	return true
}

func ReportError(r *http.Request, controller any, err error) bool {
	if err == nil {
		return false
	}
	state, ok := errorStateFromRequest(r)
	if !ok {
		return false
	}
	if controller != nil {
		state.controller = controller
	}
	state.err = err
	return true
}

func errorStateFromRequest(r *http.Request) (*requestErrorState, bool) {
	if r == nil {
		return nil, false
	}
	state, ok := r.Context().Value(requestErrorStateKey{}).(*requestErrorState)
	return state, ok && state != nil
}

func handleReportedError(ctx context.Context, w http.ResponseWriter, r *http.Request, controller any, err error) {
	logReportedError(r, err)
	ResetResponse(w)
	if handler, ok := controller.(controllerErrorHandler); ok {
		handleErr := callControllerErrorHandler(handler, w, r, err)
		if handleErr == nil {
			return
		}
		err = handleErr
		ResetResponse(w)
	}
	if DetailErrors(ctx) {
		WriteErrorDetail(w, r, err)
		return
	}
	if WriteErrorFallback(ctx, w, r) {
		return
	}
	WriteError(w, r, err)
}

func logReportedError(r *http.Request, err error) {
	if err == nil {
		return
	}
	if r == nil {
		fmt.Fprintf(os.Stderr, "lazycontroller: error: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "lazycontroller: %s %s: %v\n", r.Method, r.URL.RequestURI(), err)
}

func callControllerErrorHandler(handler controllerErrorHandler, w http.ResponseWriter, r *http.Request, err error) (handleErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			handleErr = PanicError(recovered)
		}
	}()
	return handler.HandleError(w, r, err)
}
