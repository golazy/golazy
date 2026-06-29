package lazyforms_test

import (
	"fmt"
	"strings"
	"testing/fstest"

	"golazy.dev/lazyforms"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type note struct {
	Title string
}

type noteRouter struct{}

func (noteRouter) PathForModel(model any, action string) (string, error) {
	if action == "create" {
		return "/notes", nil
	}
	return "/notes/example", nil
}

func ExampleHelpers() {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":      {Data: []byte(`{{.content}}`)},
		"notes/new.html.tpl":        {Data: []byte(`{{ form_for .Note . }}`)},
		"notes/_note_form.html.tpl": {Data: []byte(`{{ text_field "Title" }}{{ submit_button "Create" }}`)},
	})
	if err != nil {
		panic(err)
	}
	views.AddHelpers(lazyforms.Helpers(noteRouter{}))
	if err := views.Cache(); err != nil {
		panic(err)
	}

	body, err := views.RenderString(lazyview.Options{
		Variables:  map[string]any{"Note": note{Title: "Draft"}},
		Controller: "notes",
		Action:     "new",
		UseLayout:  false,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(strings.ReplaceAll(body, "\n", ""))

	// Output:
	// <form action="/notes" method="post" id="new_note" class="new_note"><label for="note_title">Title <input type="text" id="note_title" name="title" value="Draft"></label><button type="submit">Create</button></form>
}
