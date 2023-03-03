package page

/*
func ExampleLayout() {
	nodes.Beautify = true
	defer (func() {
		nodes.Beautify = false
	})()

	template := &Page{}

	template.With("hola mundo").WriteTo(os.Stdout)

	// Output:
	// <html>
	// <head>
	// </head>
	// <body>
	// hola mundo</body>
	// </html>
}
func ExampleLayout_complete() {
	nodes.Beautify = true
	defer (func() {
		nodes.Beautify = false
	})()

	template := &Page{
		Lang:     "en",
		Title:    "lazyview",
		Viewport: "width=device-width",
		Styles:   []string{"body{margin:0;padding:0;box-sizing: border-box;}"},
		Head: []any{
			Script(Async(), Src("https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js"), Crossorigin(("anonymous"))),
			Script(Type("module"),
				nodes.Raw(`import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';`),
			),
		},
		Scripts: []string{
			`document.write("hello");`,
		},
		Content: []any{"hola"},
	}

	template.With("hello").WriteTo(os.Stdout)

	// Output:
	// <html lang="en">
	// <head>
	// <title>lazyview</title>
	// <meta name="viewport" content="width=device-width"/>
	// <style>body{margin:0;padding:0;box-sizing: border-box;}</style>
	// <script>document.write("hello");</script>
	// <script async src="https://ga.jspm.io/npm:es-module-shims@1.4.6/dist/es-module-shims.js" crossorigin="anonymous"/>
	// <script type="module">import hotwiredTurbo from 'https://cdn.skypack.dev/@hotwired/turbo';</script>
	// </head>
	// <body>
	// <main>hello</main>
	// </body>
	// </html>

}

*/
