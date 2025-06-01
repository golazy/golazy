package lazyassets

import (
	"net/http"
	"testing"
)

func TestNextHandler(t *testing.T) {

	ts := NewServerTest(t)
	ts.AddFS(TestAssetsFS, "test_assets")

	ts.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("next"))
	})

	response := ts.fetch("/not_found")
	response.StatusIs(201)
	response.BodyIs("next")
}

func TestManager_ByPath(t *testing.T) {
	ts := NewServerTest(t)
	ts.AddFS(TestAssetsFS, "test_assets")

	path := "/@test/hello.world"

	response := ts.fetch(path)
	response.StatusIs(200)
	response.BodyIs("hi")
	response.EtagIs("\"8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4\"")
	response.CacheIs("")

	perm := ts.assertPermalink(path)
	response = ts.fetch(perm)
	response.StatusIs(200)
	response.BodyIs("hi")
	response.EtagIs("\"8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4\"")
	response.CacheIs("public, max-age=31536000")
}

func TestPermalinks(t *testing.T) {
	st := NewServerTest(t)
	st.AddFS(TestAssetsFS, "test_assets")

	st.permalinkIs("/@test/hello.world", "/@test/hello-8f434346648f.world")
	st.permalinkIs("/not_found", "/not_found")

	response := st.fetch(st.permalink("/@test/hello.world"))
	response.StatusIs(200)
	response.BodyIs("hi")

}
func TestManager_ETag(t *testing.T) {
	ts := NewServerTest(t)
	ts.AddFS(TestAssetsFS, "test_assets")

	response := ts.fetch("/@test/hello.world")
	response.EtagIs("\"8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4\"")

	perm := ts.assertPermalink("/@test/hello.world")
	response = ts.fetch(perm)
	response.EtagIs("\"8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4\"")

	response = ts.fetchWithReq(perm, func(req *http.Request) {
		req.Header.Set("If-None-Match", "\"8f434346648f6b96df89dda901c5176b10a6d83961dd3c1ac88b59b2dc327aa4\"")
	})
	response.StatusIs(http.StatusNotModified)
	response.CacheIs("")
}

func TestServerCSS(t *testing.T) {
	ts := NewServerTest(t)
	ts.AddFS(TestAssetsFS, "test_assets")

	response := ts.fetch("/main.css")
	response.BodyContains("bg-")

	ts.PermalinkFiles = func(f File) bool { return false }

	response = ts.fetch("/main.css")
	response.BodyContains("bg.png")

	ts.PermalinkFiles = nil
	ts.CSSTransformFiles = func(f File) bool { return false }

	response = ts.fetch("/main.css")
	response.BodyContains("bg.png")

}
