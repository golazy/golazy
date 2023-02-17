package main

import (
	"fmt"
	"os"
	"os/exec"
)

func init() {
	commands = append(commands, Command{
		Name:        "dev",
		Description: "Start development server",
		Cmd:         Dev,
	})
}

func Dev(args []string) {
	fmt.Println("This is dev")

}

type devServer struct {
	err error
}

func (d *devServer) Run() {
	state := d.Build

	for state != nil {
		state = state()
	}
}

func (d *devServer) Build() devState {

	f, err := os.CreateTemp("", "golazy-*-app")
	if err != nil {
		panic(err)
	}
	name := f.Name()
	f.Close()

	cmd := exec.Command("go", "build", "-o", name, "-tags", "dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic(err)
	}

	return nil
}

type devState func() devState
