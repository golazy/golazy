package lazyapp

import (
	"context"

	"golazy.dev/lazyjobs"
)

// JobsConfig initializes lazyjobs with the dependency-initialized app context.
type JobsConfig func(context.Context) (lazyjobs.Config, error)

// Jobs adapts a static lazyjobs.Config for Config.Jobs.
func Jobs(config lazyjobs.Config) JobsConfig {
	return func(context.Context) (lazyjobs.Config, error) {
		return config, nil
	}
}
