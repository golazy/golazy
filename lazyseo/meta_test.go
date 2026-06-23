package lazyseo_test

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"golazy.dev/lazyseo"
	"golazy.dev/lazyseo/jsonld"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

type controller struct {
	data map[string]any
}

func (c *controller) Set(name string, value any) {
	if c.data == nil {
		c.data = map[string]any{}
	}
	c.data[name] = value
}

func TestSetStoresMetaForSEOHelper(t *testing.T) {
	c := &controller{}
	updated := time.Date(2026, 6, 20, 11, 12, 13, 0, time.UTC)
	lazyseo.Set(c,
		lazyseo.Title(`Hello <Go>`),
		lazyseo.Description(`Build "boring" web apps.`),
		lazyseo.Language("de"),
		lazyseo.Canonical("https://golazy.dev/posts/hello"),
		lazyseo.AlternateURL("en", "https://golazy.dev/en/posts/hello"),
		lazyseo.AlternateLink(lazyseo.Alternate{
			Media: "(max-width: 640px)",
			URL:   "https://m.golazy.dev/posts/hello",
		}),
		lazyseo.URL("https://golazy.dev/posts/hello"),
		lazyseo.Image("https://golazy.dev/preview.png"),
		lazyseo.ImageAlt("Preview of Hello <Go>"),
		lazyseo.OpenGraphData(lazyseo.OpenGraph{ImageWidth: 1200, ImageHeight: 630}),
		lazyseo.Kind(lazyseo.Article),
		lazyseo.JSONLD(jsonld.NewArticle("Hello <Go>")),
		lazyseo.PublishedTime(updated.Add(-24*time.Hour)),
		lazyseo.UpdatedTime(updated),
	)

	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<head>{{seo}}</head><main>{{.content}}</main>`)},
		"posts/show.html.tpl":  {Data: []byte(`<p>{{.body}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(lazyseo.Helpers(
		lazyseo.SiteName("GoLazy"),
		lazyseo.Locale("en_US"),
		lazyseo.TwitterCardType("summary_large_image"),
	))

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  c.data,
		Controller: "posts",
		Action:     "show",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	body := out.String()
	assertContains(t, body, `<title>Hello &lt;Go&gt; - GoLazy</title>`)
	assertContains(t, body, `<meta name="description" content="Build &#34;boring&#34; web apps.">`)
	assertContains(t, body, `<link rel="canonical" href="https://golazy.dev/posts/hello">`)
	assertContains(t, body, `<link rel="alternate" hreflang="en" href="https://golazy.dev/en/posts/hello">`)
	assertContains(t, body, `<link rel="alternate" media="(max-width: 640px)" href="https://m.golazy.dev/posts/hello">`)
	assertContains(t, body, `<meta property="og:title" content="Hello &lt;Go&gt; - GoLazy">`)
	assertContains(t, body, `<meta property="og:url" content="https://golazy.dev/posts/hello">`)
	assertContains(t, body, `<meta property="og:image" content="https://golazy.dev/preview.png">`)
	assertContains(t, body, `<meta property="og:image:secure_url" content="https://golazy.dev/preview.png">`)
	assertContains(t, body, `<meta property="og:image:width" content="1200">`)
	assertContains(t, body, `<meta property="og:image:height" content="630">`)
	assertContains(t, body, `<meta property="og:image:alt" content="Preview of Hello &lt;Go&gt;">`)
	assertContains(t, body, `<meta property="og:type" content="article">`)
	assertContains(t, body, `<meta property="og:locale" content="en_US">`)
	assertContains(t, body, `<meta name="twitter:card" content="summary_large_image">`)
	assertContains(t, body, `<meta name="twitter:image:alt" content="Preview of Hello &lt;Go&gt;">`)
	assertContains(t, body, `<meta property="article:published_time" content="2026-06-19T11:12:13Z">`)
	assertContains(t, body, `<meta property="article:modified_time" content="2026-06-20T11:12:13Z">`)
	assertContains(t, body, `<script type="application/ld+json">{"@context":"https://schema.org","@type":"Article","headline":"Hello \u003cGo\u003e"}</script>`)
}

func TestLanguageHelperUsesRequestMetaAndDefaults(t *testing.T) {
	c := &controller{}
	lazyseo.Set(c, lazyseo.Language(`pt-BR`))

	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<html lang="{{seo_lang}}">{{.content}}</html>`)},
		"home/index.html.tpl":  {Data: []byte(`home`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(lazyseo.Helpers(lazyseo.Language("en")))

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Variables:  c.data,
		Controller: "home",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out.String(), `<html lang="pt-BR">home</html>`)
}

func TestSEOHelperUsesDefaultsWithoutRequestMeta(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<html lang="{{seo_lang}}"><head>{{seo}}</head><main>{{.content}}</main></html>`)},
		"home/index.html.tpl":  {Data: []byte(`home`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(lazyseo.Helpers(
		lazyseo.Title("Home"),
		lazyseo.SiteName("GoLazy"),
		lazyseo.Language("en"),
		lazyseo.Canonical("https://golazy.dev/"),
	))

	var out strings.Builder
	err = views.Render(lazyview.Options{
		Writer:     &out,
		Controller: "home",
		Action:     "index",
		UseLayout:  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertContains(t, out.String(), `<title>Home - GoLazy</title>`)
	assertContains(t, out.String(), `<html lang="en">`)
	assertContains(t, out.String(), `<link rel="canonical" href="https://golazy.dev/">`)
	assertContains(t, out.String(), `<meta property="og:url" content="https://golazy.dev/">`)
}

func TestSEOHelperKeepsCompleteTitles(t *testing.T) {
	for _, test := range []struct {
		name  string
		title string
		want  string
	}{
		{name: "site title", title: "golazy.dev", want: `<title>golazy.dev</title>`},
		{name: "already qualified", title: "Views | GoLazy Guides", want: `<title>Views | GoLazy Guides</title>`},
		{name: "plain", title: "Views", want: `<title>Views - golazy.dev</title>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &controller{}
			lazyseo.Set(c, lazyseo.Title(test.title))
			views, err := lazyview.New(fstest.MapFS{
				"layouts/app.html.tpl": {Data: []byte(`<head>{{seo}}</head><main>{{.content}}</main>`)},
				"home/index.html.tpl":  {Data: []byte(`home`)},
			})
			if err != nil {
				t.Fatal(err)
			}
			views.AddHelpers(lazyseo.Helpers(lazyseo.SiteName("golazy.dev")))

			var out strings.Builder
			err = views.Render(lazyview.Options{
				Writer:     &out,
				Variables:  c.data,
				Controller: "home",
				Action:     "index",
				UseLayout:  true,
			})
			if err != nil {
				t.Fatal(err)
			}
			assertContains(t, out.String(), test.want)
		})
	}
}

func assertContains(t *testing.T, body, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Fatalf("body does not contain %q:\n%s", expected, body)
	}
}
