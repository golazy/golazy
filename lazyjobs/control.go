package lazyjobs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golazy.dev/lazycontrolplane"
)

const ControlJobsPath = "/jobs"

func RegisterControlPlaneHandlers(controlPlane *lazycontrolplane.ControlPlane, runner *JobRunner) {
	if controlPlane == nil || runner == nil {
		return
	}
	if controlPlane.HandlesPath(ControlJobsPath) {
		return
	}
	controlPlane.Handle("GET "+ControlJobsPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		snapshot, err := runner.Snapshot(r.Context())
		if err != nil {
			http.Error(w, fmt.Sprintf("jobs: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			http.Error(w, fmt.Sprintf("jobs: %v", err), http.StatusInternalServerError)
		}
	}))
}

func RegisterLazyDevHandlers(controlPlane *lazycontrolplane.ControlPlane, runner *JobRunner) {
	RegisterControlPlaneHandlers(controlPlane, runner)
}
