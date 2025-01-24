package middlewares

import (
	"net/http"

	"golazy.dev/lazysupport"
)

const FormMethodMiddleware = "form_method"

var FormMethod = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Be aware that FormValue will read the request body
		method := r.FormValue("_method")
		if validFormMethods.Has(method) {
			r.Method = method
		}
	}
})

var validFormMethods = lazysupport.NewStringSet("PUT", "PATCH", "DELETE")
