package cli

import (
	"embed"
	_ "embed"
)

//go:embed all:project
var projectTemplate embed.FS
