package layout

import (
	"os"

	. "github.com/guillermo/golazy/lazyview/html"
	"github.com/guillermo/golazy/lazyview/nodes"
)

func ExampleComponent() {

	nodes.Beautify = true
	defer (func() {
		nodes.Beautify = false
	})()

	template := &LayoutTemplate{}
	template.AddComponent(Component{
		Scripts: []string{`document.Write("hello world");`},
		Styles:  []string{`body{background: red;}`},
		Head: []interface{}{
			Script(Type("module"), Src("https://google.com/s.rs")),
		},
	})

	template.With("hola mundo").WriteTo(os.Stdout)

	// Output:
	// <html>
	// <head>
	// <style>body{background: red;}</style>
	// <script>document.Write("hello world");</script>
	// <script type="module" src="https://google.com/s.rs"/>
	// </head>
	// <body>
	// hola mundo</body>
	// </html>

}
