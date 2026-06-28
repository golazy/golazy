package lazymigrate

import "context"

type Backend interface {
	Setup(context.Context) error
	List(context.Context) ([]BackendMigration, error)
	Run(context.Context, Step) error
	DumpSchema(context.Context) ([]byte, error)
	LoadSchema(context.Context, []byte) error
}
