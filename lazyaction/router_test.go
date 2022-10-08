package lazyaction

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedirect(t *testing.T) {

	router := Router(Routes{
		Prefix{"posts", Routes{
			RedirectPath{To: "../posts"},
		}},
	})

	testRoute := func(path, expectation string) {
		t.Helper()
		r := httptest.NewRequest("", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if strings.TrimSpace(w.Body.String()) != expectation {
			t.Errorf("Expecting %q to send %q. Got %q", path, expectation, w.Body.String())
		}
	}

	testRoute("/posts/", "<a href=\"/posts\">Permanent Redirect</a>.")

}

func TestRouter(t *testing.T) {

	say := func(what string) Handler {
		return HandlerFunc(func(w ResponseWriter, r *Request) {
			w.Write([]byte(fmt.Sprint(what, r.Params)))
		})
	}

	router := &Router{
		Path{"pizza", "", "", say("pizza")},
		CatchAllPrefix{"page_id", Routes{
			Path{"", "", "", say("show_page")},
			Path{"publish", "", "", say("publish_page")},
		}},
		Path{"posts", "", "", say("posts_index")},
		Prefix{"posts", Routes{
			CatchAllPath{"post_id", "", "", say("post_show")},
			Path{"", "", "", say("post index")},
			Path{"new", "", "", say("post new")},
			CatchAllPrefix{"post_id", Routes{
				Path{"publish", "", "", say("publish")},
			}},
			Path{"publish", "", "", say("publish")},
		}},
	}

	testRoute := func(path, expectation string) {
		t.Helper()
		r := httptest.NewRequest("", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if strings.TrimSpace(w.Body.String()) != expectation {
			t.Errorf("Expecting %q to send %q. Got %q", path, expectation, w.Body.String())
		}
	}

	testRoute("/posts/33/publish.json?hola=mundo", "publishmap[format:[json] post_id:[33]]")
	testRoute("/", "Not Found")
	testRoute("/posts", "posts_indexmap[]")
	testRoute("/posts/33", "post_showmap[post_id:[33]]")
	testRoute("/posts/33.json", "post_showmap[format:[json] post_id:[33]]")
	testRoute("/posts/new", "post newmap[]")
	testRoute("/what", "Not Found")
}

func ExampleRoutes() {

	say := func(what string) HandlerFunc {
		return func(w ResponseWriter, r *Request) {
			params, _ := json.Marshal(r.Params)
			w.Write([]byte(fmt.Sprintf("%s Params: %s", what, params)))
		}
	}

	router := &Router{
		Path{"", "GET", "", say("Home page")},
		Path{"pages", "", "", say("Pages index")}, // HTTP Medoth defaults to GET
		Prefix{"pages", Routes{ // Handles `pages/`
			RedirectPath{
				Path: "",
				To:   "../pages",
			},
			CatchAllPath{"page_id", "", "", say("Nice Page")}, // Matches `/pages/:page_id` assiging `page_id` to httpRequest.Form.Values.Get("page_id")
			CatchAllPrefix{"page_id", Routes{
				Path{"share", "POST", "", say("Page shared!")}, // Matches `POST /pages/:page_id/share`
				Prefix{"paragraphs", Routes{
					CatchAllPath{"paragraph_id", "", "", say("Paragraph")},
				}},
			}},
		}},
	}

	// For testing
	query := func(path string) string {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", path, nil)
		router.ServeHTTP(w, r)
		return w.Body.String()

	}

	fmt.Println(query("/pages/33"))
	fmt.Println(query("/pages/33/paragraph/42.json"))
	// Output:
	// Nice Page Params: {"page_id":["33"]}
	// Paragraph Params: {"format":["json"],"page_id":["33"],"paragraph_id":["42"]}
}
