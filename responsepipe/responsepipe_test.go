package responsepipe

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResponsepipe(t *testing.T) {

	handler := func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			reader, writer, code, err := New(w, r, h)
			if err != nil {
				return
			}

			io.Copy(writer, reader)

			data, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}

			writer.WriteHeader()

			_, err = writer.Write([]byte(strings.ToUpper(string(data))))
			if err != nil {
				t.Fatal(err)
			}

		})
	}

}

func Test