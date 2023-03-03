package fallback

import "golazy.dev/lazyapp"

var App = &lazyapp.App{
	Name: "fallback",
}

func init() {
	App.Init()
}
