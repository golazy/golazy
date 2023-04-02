package base

import (
	"embed"

	"golazy.dev/lazydev/generator"
)

//go:embed all:project
var fs embed.FS

var Project = generator.FSGenerator{
	FS:         fs,
	TrimPrefix: "project",
}
