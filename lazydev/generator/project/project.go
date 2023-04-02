package project

import (
	"embed"

	"golazy.dev/lazydev/generator"
)

//go:embed app
var appFS embed.FS

var App = &generator.FSGenerator{
	FS:           appFS,
	TrimPrefix:   "app",
	RequiredVars: []string{"App"},
}

//go:embed base
var baseFS embed.FS

var Base = &generator.FSGenerator{
	FS:           baseFS,
	TrimPrefix:   "base",
	RequiredVars: []string{"GoVersion"},
}
