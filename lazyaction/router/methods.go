package router

var Methods = []string{"GET", "POST", "OPTIONS", "PUT", "PATCH", "DELETE", "WS"}

func IsMethod(method string) int {
	for i, m := range Methods {
		if m == method {
			return i
		}
	}
	return -1

}
