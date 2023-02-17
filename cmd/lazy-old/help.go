package main

import "fmt"

func init() {
	commands = append(commands, Command{
		Name:        "help",
		Description: "Show help",
		Cmd:         Help,
	})
}

func Help([]string) {
	fmt.Println("This is help")
}
