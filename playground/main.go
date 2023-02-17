package main

import "strconv"

type App struct {
	Scenes map[string]*Scene
}

type Scene struct {
	Css    string
	Events map[string][]func()
}

func main() {

	app := NewApp()

	home := app.NewScene("Setup")

	container := home.Add("flexbox")
	container.Add("h1", "List of mails")
	main := container.Add("flexbox", "flex-direction: row; justify-content: space-between;")
	mailboxes := main.Add("div", "Mailboxes")
	mailboxes.Content("Loading mailboxes...")

	mailboxes.On("load", func() {

		ul := mailboxes.Content("ul")
		for i := 0; i < 10; i++ {

			li := ul.Add("li", "Mailbox "+strconv.Itoa(i))
			li.On("click", func() {
				home.Set("mailbox", i)
			})

			if home.Get("mailbox") == i {
				li.Set("class", "selected")
			}
		}
	})
	app.Open()

}
