package lazyapp

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"golazy.dev/lazycache"
	"golazy.dev/lazycache/inmemorycache"
	"golazy.dev/lazyturbo"
	"golazy.dev/lazyview"
	_ "golazy.dev/lazyview/gotmpl"
)

func testCacheContext(t *testing.T) context.Context {
	t.Helper()
	backend, err := inmemorycache.New(inmemorycache.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cache, err := lazycache.New(lazycache.Options{Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	return lazycache.WithBuildVersion(lazycache.WithCache(context.Background(), cache), "testbuild")
}

func TestCacheHelperCachesPartialBody(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl": {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl": {Data: []byte(`{{ cache "post" "card" . }}`)},
		"posts/_card.html.tpl": {Data: []byte(`<p>{{inc}} {{.name}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	views.AddHelpers(cacheHelpers())
	views.AddHelpers(map[string]any{
		"inc": func() int {
			count++
			return count
		},
	})
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	ctx := testCacheContext(t)
	var first strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &first,
		Variables:  map[string]any{"name": "Ada"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}
	var second strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &second,
		Variables:  map[string]any{"name": "Grace"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}

	if got, want := first.String(), `<main><p>1 Ada</p></main>`; got != want {
		t.Fatalf("first render = %q, want %q", got, want)
	}
	if got, want := second.String(), first.String(); got != want {
		t.Fatalf("second render = %q, want cached %q", got, want)
	}
	if count != 1 {
		t.Fatalf("partial render count = %d, want 1", count)
	}
}

func TestCacheFullHelpersShareExplicitKey(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":    {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl":    {Data: []byte(`{{ cache (cache_key "post" .id) "card" . }}`)},
		"articles/index.html.tpl": {Data: []byte(`{{ cachef "post" .id "card" . }}`)},
		"posts/_card.html.tpl":    {Data: []byte(`<p>{{.name}}</p>`)},
		"articles/_card.html.tpl": {Data: []byte(`<article>{{.name}}</article>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(cacheHelpers())
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	ctx := testCacheContext(t)
	var first strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &first,
		Variables:  map[string]any{"id": 1, "name": "Ada"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}
	var second strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &second,
		Variables:  map[string]any{"id": 1, "name": "Grace"},
		Controller: "articles",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}

	if got, want := first.String(), `<main><p>Ada</p></main>`; got != want {
		t.Fatalf("first render = %q, want %q", got, want)
	}
	if got, want := second.String(), `<main><p>Ada</p></main>`; got != want {
		t.Fatalf("second render = %q, want shared cached body %q", got, want)
	}
}

func TestCacheHelpersIncludeVariantAndBuildVersion(t *testing.T) {
	ctx := &lazyview.Context{
		Context:    testCacheContext(t),
		Namespace:  "admin",
		Controller: "posts",
		Action:     "show",
		Format:     "html",
		Variants:   []string{"mobile"},
	}

	key, err := partialCacheKey(ctx, "post", "card")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := key, "build-testbuild-variant-mobile-admin-posts-show-html-card-post"; got != want {
		t.Fatalf("partial cache key = %q, want %q", got, want)
	}

	key, err = partialCacheKey(ctx, templateCacheKey("post-1"), "card")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := key, "build-testbuild-variant-mobile-post-1"; got != want {
		t.Fatalf("explicit cache key = %q, want %q", got, want)
	}
}

func TestTurboFrameUsesCacheKeyForBodyOnly(t *testing.T) {
	views, err := lazyview.New(fstest.MapFS{
		"layouts/app.html.tpl":       {Data: []byte(`<main>{{.content}}</main>`)},
		"posts/index.html.tpl":       {Data: []byte(`{{ turbo_frame "post" . (cache_key "post" .id) (turbo_src .src) }}`)},
		"posts/_post_frame.html.tpl": {Data: []byte(`<p>{{.name}}</p>`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	views.AddHelpers(lazyturbo.Helpers())
	views.AddHelpers(cacheHelpers())
	if err := views.Cache(); err != nil {
		t.Fatal(err)
	}

	ctx := testCacheContext(t)
	var first strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &first,
		Variables:  map[string]any{"id": 1, "name": "Ada", "src": "/first"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}
	var second strings.Builder
	if err := views.Render(lazyview.Options{
		Context:    ctx,
		Writer:     &second,
		Variables:  map[string]any{"id": 1, "name": "Grace", "src": "/second"},
		Controller: "posts",
		Action:     "index",
		UseLayout:  true,
	}); err != nil {
		t.Fatal(err)
	}

	if got, want := first.String(), `<main><turbo-frame id="post" src="/first"><p>Ada</p></turbo-frame></main>`; got != want {
		t.Fatalf("first render = %q, want %q", got, want)
	}
	if got, want := second.String(), `<main><turbo-frame id="post" src="/second"><p>Ada</p></turbo-frame></main>`; got != want {
		t.Fatalf("second render = %q, want cached body with current attrs %q", got, want)
	}
}
