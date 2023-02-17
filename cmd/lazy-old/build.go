package main

import "fmt"

func init() {
	commands = append(commands, Command{
		Name:        "build",
		Description: "Build production binary",
		Cmd:         Production,
	})
}

func Production([]string) {
	fmt.Println("This is production")
}
