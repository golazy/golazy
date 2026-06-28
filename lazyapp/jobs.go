package lazyapp

import (
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazyjobs"
)

func jobsControlPlane(controlPlane *lazycontrolplane.ControlPlane, jobs *lazyjobs.JobRunner) *lazycontrolplane.ControlPlane {
	if jobs == nil {
		return controlPlane
	}
	if controlPlane == nil {
		controlPlane = lazycontrolplane.New(lazycontrolplane.Config{})
	}
	lazyjobs.RegisterControlPlaneHandlers(controlPlane, jobs)
	return controlPlane
}
