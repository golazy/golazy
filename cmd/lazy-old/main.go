package main

import (
	"fmt"
	"os"
	"path"

	"golazy.dev/lazysupport"
)

var commands []Command

var program string // os.Args[0]
var command string // os.Args[1]
var args []string  // os.Args[2:]

func init() {
	program = os.Args[0]
	if len(os.Args) == 1 {
		return
	}
	command = os.Args[1]
	if len(os.Args) == 2 {
		return
	}
	args = os.Args[2:]
}

type Command struct {
	Name        string
	Description string
	Help        string
	Cmd         func(args []string)
}

func RunCommand(name string, args []string) {
	for _, cmd := range commands {
		if cmd.Name == name {
			cmd.Cmd(args)
			return
		}
	}

	fmt.Println("Unknown command:", name)
}

func ListCommands() {
	t := lazysupport.Table{}

	for _, cmd := range commands {
		t.Values = append(t.Values, []string{cmd.Name, cmd.Description})
	}

	fmt.Println(t.String())

}

func main() {
	if command == "" {
		fmt.Println("Usage:", path.Base(program), "COMMAND", "[ARGS]", "\n")
		ListCommands()
		return
	}
	RunCommand(command, args)
}
