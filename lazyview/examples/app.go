package main

import (
	. "github.com/guillermo/lazyview/html"
	"github.com/guillermo/lazyview/serve"
)

func main() {

	DefineComponent(
		Use("https://unpkg.com/@hotwired/stimulus@3.0.1/dist/stimulus.js"),
		Script(`
		// hello_controller.js
		import { Controller } from "stimulus"

		export default class extends Controller {
			static targets = [ "name", "output" ]

			greet() {
				this.outputTarget.textContent =
				`+"`Hello, ${this.nameTarget.value}!`"+`
			}
		}
		`),
		Raw(`<div data-controller="hello">
		<input data-hello-target="name" type="text">
	  
		<button data-action="click->hello#greet">
		  Greet
		</button>
	  
		<span data-hello-target="output">
		</span>
	  </div>`),
	)

	serve.ServePage(Html(
		Head(
			Title("hola"),
		),
		Body(
			H1("hola"),
		),
	),
	)

}
