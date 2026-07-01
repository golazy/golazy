package lazymigrate

import "context"

// Backend connects lazymigrate plans to a concrete store.
//
// Production implementations must be safe for multiple application processes to
// call at the same time. Migration application must be synchronized by the
// backend so only one caller can apply a conflicting step or plan, the schema
// change and metadata update are committed atomically, and a concurrent caller
// that races with an already-applied migration does not corrupt migration
// state.
type Backend interface {
	Setup(context.Context) error
	List(context.Context) ([]BackendMigration, error)
	Run(context.Context, Step) error
	DumpSchema(context.Context) ([]byte, error)
	LoadSchema(context.Context, []byte) error
}
