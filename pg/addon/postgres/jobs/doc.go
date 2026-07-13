// Package jobs registers the PostgreSQL-backed jobs add-on for GoLazy apps.
//
// Importing the package also imports the base postgres add-on. Selecting
// "postgres/jobs" therefore resolves "postgres" first, mounts the pgjobs
// migrations, and configures the durable lazyjobs backend.
package jobs
