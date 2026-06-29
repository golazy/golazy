package lazysession_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"golazy.dev/lazysession"
)

func ExampleManager() {
	manager, err := lazysession.NewManager(lazysession.Config{
		Name: "visits",
		Key:  "example-session-secret",
		Options: &lazysession.Options{
			Path:     "/",
			MaxAge:   3600,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
	})
	if err != nil {
		panic(err)
	}

	handler := manager.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := lazysession.Get(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		visits, _ := session.Values["visits"].(int)
		visits++
		session.Values["visits"] = visits

		fmt.Fprintf(w, "visit %d\n", visits)
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	fmt.Print(first.Body.String())

	secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	secondRequest.AddCookie(first.Result().Cookies()[0])
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, secondRequest)
	fmt.Print(second.Body.String())

	// Output:
	// visit 1
	// visit 2
}
