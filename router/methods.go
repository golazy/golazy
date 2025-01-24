package router

var methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}

func isMethod(method string) int {
	for i, m := range methods {
		if m == method {
			return i
		}
	}
	return -1

}
