package lazyapp

import (
	"context"

	"golazy.dev/lazyaddon"
	"golazy.dev/lazycontrolplane"
	"golazy.dev/lazydeps"
	"golazy.dev/lazyfs"
	"golazy.dev/lazyjobs"
	"golazy.dev/lazymigrate"
	"golazy.dev/lazyroutes"
)

// FilesEvent lets selected add-ons contribute view and public filesystem
// layers. lazyapp adds framework layers before this hook and application layers
// after it, then seals both filesystems.
type FilesEvent struct {
	Context context.Context
	Views   *lazyfs.FS
	Public  *lazyfs.FS
	Addons  *lazyaddon.Scope
}

// DependenciesEvent lets add-ons initialize services in the app dependency
// scope.
type DependenciesEvent struct {
	Context      context.Context
	Dependencies *lazydeps.Scope
	Addons       *lazyaddon.Scope
}

// MigrationsEvent lets add-ons provide database backends and migration
// filesystems after dependencies are available.
type MigrationsEvent struct {
	Context   context.Context
	Databases *lazymigrate.Databases
	Catalog   *lazymigrate.Catalog
	Addons    *lazyaddon.Scope
}

// JobsEvent lets an add-on enable or refine the application's jobs runner.
type JobsEvent struct {
	Context context.Context
	Config  lazyjobs.Config
	Enabled bool
	Addons  *lazyaddon.Scope
}

// RoutesEvent lets add-ons draw routes after the application drawer.
type RoutesEvent struct {
	Context context.Context
	Router  *lazyroutes.Scope
	Addons  *lazyaddon.Scope
}

// HelpersEvent lets add-ons register view helpers before application-owned
// helpers are applied.
type HelpersEvent struct {
	Context context.Context
	Helpers *Helpers
	Addons  *lazyaddon.Scope
}

// Add appends one helper map to the event.
func (event *HelpersEvent) Add(helpers map[string]any) {
	if event == nil || event.Helpers == nil || len(helpers) == 0 {
		return
	}
	*event.Helpers = append(*event.Helpers, helpers)
}

// ControlPlaneEvent lets add-ons register owned operational endpoints,
// readiness checks, and development-panel descriptors. Build-tagged callback
// files determine which registrations exist in lazydev and production builds.
type ControlPlaneEvent struct {
	Context      context.Context
	ControlPlane lazycontrolplane.Registrar
	Addons       *lazyaddon.Scope
}

var (
	// FilesHook owns view and public filesystem contributions.
	FilesHook = lazyaddon.DefineHook[FilesEvent]("golazy.dev/lazyapp/files", 1)
	// DependenciesHook owns dependency service contributions.
	DependenciesHook = lazyaddon.DefineHook[DependenciesEvent]("golazy.dev/lazyapp/dependencies", 1)
	// MigrationsHook owns database and migration contributions.
	MigrationsHook = lazyaddon.DefineHook[MigrationsEvent]("golazy.dev/lazyapp/migrations", 1)
	// JobsHook owns jobs runner contributions.
	JobsHook = lazyaddon.DefineHook[JobsEvent]("golazy.dev/lazyapp/jobs", 1)
	// RoutesHook owns application route contributions.
	RoutesHook = lazyaddon.DefineHook[RoutesEvent]("golazy.dev/lazyapp/routes", 1)
	// HelpersHook owns template helper contributions.
	HelpersHook = lazyaddon.DefineHook[HelpersEvent]("golazy.dev/lazyapp/helpers", 1)
	// ControlPlaneHook owns operational control-plane contributions.
	ControlPlaneHook = lazyaddon.DefineHook[ControlPlaneEvent]("golazy.dev/lazyapp/controlplane", 1)
)

func runAddonHook[T any](scope *lazyaddon.Scope, hook lazyaddon.Hook[T], event *T) {
	if err := lazyaddon.Run(scope, hook, event); err != nil {
		panic(err)
	}
}
