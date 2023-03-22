package command_test

import (
	"testing"

	"golazy.dev/lazydev/command"
)

func TestCommand(t *testing.T) {
	command.Add(command.Command{
		Use:  "dev",
		Desc: "Start development server",
		Long: `Start development server`,
		Flags: []command.Flag[any]{
			command.Flag[bool]{
				Long:        "noportal",
				Name:        "n",
				Env:         "LAZYDEV_NO_PORTAL",
				Default:     false,
				Description: "Disable portal and serve only the main package",
			},
		},
	})

}
