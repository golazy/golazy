package main

func init() {
	commands = append(commands, Command{
		Name:        "routes",
		Description: "Display App Routes",
		Cmd:         Routes,
	})
}

func Routes([]string) {

}
