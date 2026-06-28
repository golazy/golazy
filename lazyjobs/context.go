package lazyjobs

import "context"

type contextKey struct{}

func WithRunner(ctx context.Context, runner *JobRunner) context.Context {
	return context.WithValue(ctx, contextKey{}, runner)
}

func RunnerFromContext(ctx context.Context) (*JobRunner, bool) {
	runner, ok := ctx.Value(contextKey{}).(*JobRunner)
	return runner, ok
}
