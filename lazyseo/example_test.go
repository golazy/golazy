package lazyseo_test

import (
	"fmt"
	"strings"
	"testing/fstest"

	"golazy.dev/lazyseo"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func Example() {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<html lang="{{seo_lang}}"><head>{{seo}}</head><main>{{.content}}</main></html>`)},
		"pages/show.html.tpl":  {Data: []byte(`Hello`)},
	})
	if err != nil {
		panic(err)
	}
	views.AddHelpers(lazyseo.Helpers(
		lazyseo.SiteName("Example"),
		lazyseo.Language("en"),
	))

	meta := lazyseo.New(
		lazyseo.Title("About"),
		lazyseo.Description("About the example site."),
		lazyseo.Canonical("https://example.com/about"),
		lazyseo.Kind(lazyseo.WebPage),
	)

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  map[string]any{"seo": meta},
		Controller: "pages",
		Action:     "show",
		UseLayout:  true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(out.String())

	// Output:
	// <html lang="en"><head><title>About - Example</title>
	// <meta name="description" content="About the example site.">
	// <link rel="canonical" href="https://example.com/about">
	// <meta property="og:title" content="About - Example">
	// <meta property="og:description" content="About the example site.">
	// <meta property="og:site_name" content="Example">
	// <meta property="og:url" content="https://example.com/about">
	// <meta property="og:type" content="website">
	// <meta name="twitter:title" content="About - Example">
	// <meta name="twitter:description" content="About the example site.">
	// </head><main>Hello</main></html>
}
