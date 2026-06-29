package lazysse_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"golazy.dev/lazysse"
)

func ExampleServe() {
	request := httptest.NewRequest(http.MethodGet, "/events", nil)
	response := httptest.NewRecorder()

	err := lazysse.Serve(response, request, func(stream *lazysse.Stream) error {
		return stream.Send(lazysse.Event{
			Event: "message",
			ID:    "1",
			Data:  []string{"hello"},
		})
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(response.Header().Get("Content-Type"))
	fmt.Print(response.Body.String())

	// Output:
	// text/event-stream
	// event: message
	// id: 1
	// data: hello
	//
}
