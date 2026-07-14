// Package jobs registers the PostgreSQL-backed jobs add-on for GoLazy apps.
//
// Most applications install it from the project root:
//
//	lazy add postgres/jobs
//
// The add-on requires and selects the base postgres add-on automatically.
// Applications that construct their selection manually can use:
//
//	app := lazyapp.New(lazyapp.Config{
//		Addons: lazyaddon.Select(jobs.AddonID),
//		Jobs: lazyapp.Jobs(lazyjobs.Config{
//			Workers: 4,
//		}),
//	})
//
// The add-on mounts the pgjobs migrations and supplies the durable backend
// while preserving application-owned settings such as workers, queues, and
// job definitions. After lazyapp.New returns, another selected add-on can
// obtain the backend with:
//
//	backend, err := lazyaddon.Require(app.Addons, jobs.BackendCapability)
package jobs
