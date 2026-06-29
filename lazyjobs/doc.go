// Package lazyjobs defines typed background jobs, a runner, and the storage
// contract used to persist queued work.
//
// A job is a pointer to a struct that implements Job. The struct is the payload:
// JobRunner.Enqueue JSON-encodes it, stores that JSON through a Backend, and a
// worker later decodes the record back into the registered concrete type before
// calling Work. Job.Kind is the stable name stored with each record, so it should
// not be changed casually once jobs have been queued.
//
// JobRunner.Register records the job definitions the runner is allowed to
// decode and run. The optional QueueNamer and RetryPolicy interfaces let a job
// choose a queue, max attempts, and retry delay. Embedding BaseJob is the common
// way to use the default queue, 25 attempts, and an exponential retry delay
// capped at one minute while overriding only the methods a job needs.
//
// The runner owns worker lifecycle and state transitions. Enqueued records start
// as StatePending. Backend.Claim moves due work on a watched queue to
// StateRunning and increments the attempt count. A successful Work call moves the
// record to StateSucceeded. A returned error or panic moves the record to
// StateRetrying with a future RunAt until attempts are exhausted, then the
// record moves to StateDiscarded with LastError. Decode failures and unknown job
// kinds are discarded because the runner cannot safely execute them.
//
// Backend is deliberately small: it persists records, atomically claims due
// work, records completion, retry, and discard decisions, and exposes List and
// Stats for inspection. The inmemoryjobs backend is useful for development,
// tests, and apps that do not need durable jobs. Durable applications can supply
// another backend without changing job types or runner code.
//
// In a GoLazy application, lazyapp.Config.Jobs is the usual integration point.
// lazyapp calls the JobsConfig after lazydeps has initialized the app context,
// fills in inmemoryjobs.New when no backend is provided, creates the JobRunner,
// stores it in the context with WithRunner, starts it, and registers the job
// control-plane endpoint. Code that receives an app request context can call
// RunnerFromContext and enqueue work without importing the application package.
//
// RegisterControlPlaneHandlers exposes GET /jobs on a lazycontrolplane.ControlPlane
// and returns a Snapshot with definitions, aggregate stats, and recent records.
// RegisterLazyDevHandlers currently registers the same endpoint; lazyapp calls it
// for lazydev builds as part of the development control plane. Custom apps only
// need to call these functions when they assemble lazycontrolplane and JobRunner
// directly instead of using lazyapp.
package lazyjobs
