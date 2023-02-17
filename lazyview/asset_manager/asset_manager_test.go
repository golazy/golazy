package asset_manager

import (
	"bytes"
	"embed"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
)

//go:embed **/*.sample
var sampleFiles embed.FS

// tR testRequest
func tR(t *testing.T, am *AssetManager, path string, code int, responseBody []byte, requestH, responseH http.Header) {
	t.Helper()
	r := httptest.NewRequest("GET", path, nil)
	for k, v := range requestH {
		r.Header[k] = v
	}

	w := httptest.NewRecorder()
	am.ServeHTTP(w, r)

	// Check resposne headers
	for k, v := range responseH {
		if w.Header().Get(k) != v[0] {
			t.Errorf("expected header %s to be %s, got %s", k, v[0], w.Header().Get(k))
		}
	}

	// Check response code
	if w.Code != code {
		t.Errorf("expected code %d, got %d", code, w.Code)
	}

	// Check resposne code
	if !bytes.Equal(w.Body.Bytes(), responseBody) {
		t.Errorf("expected body %s, got %s", string(responseBody), w.Body.String())
	}
}

func compress(s string) []byte {
	buf := new(bytes.Buffer)
	comp := brotli.NewWriterLevel(buf, brotli.BestCompression)
	comp.Write([]byte(s))
	comp.Close()
	return buf.Bytes()
}

func TestAssetManager(t *testing.T) {
	am := &AssetManager{Prefix: "assets"}

	am.Add("str.asset", "string")

	tR(t, am, am.FullPath("str.asset"), 200, []byte("string"), nil, http.Header{"Etag": {"\"473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8\""}})
	tR(t, am,
		am.Permalink("str.asset"),
		200,
		[]byte("string"),
		nil,
		http.Header{
			"Etag":          {"\"473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8\""},
			"Cache-Control": {"max-age=31536000"},
		},
	)
	// Test etags
	tR(t, am,
		"/assets/str.asset",
		304,
		[]byte{},
		http.Header{
			"If-None-Match": {"\"473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8\""},
		},
		http.Header{
			"Etag": {"\"473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8\""},
		},
	)

	// Test Compression
	tR(t, am,
		"/assets/str.asset",
		200,
		compress("string"),
		http.Header{
			"Accept-Encoding": {"br"},
		},
		http.Header{
			"Etag":             {"\"473287f8298dba7163a897908958f7c0eae733e25d2e027992ea2edc9bed2fa8\""},
			"Content-Encoding": {"br"},
		},
	)

	am.AddReader("readerAsset", strings.NewReader("reader"))
	tR(t, am, "/assets/readerAsset", 200, []byte("reader"), nil, http.Header{"Etag": {"\"3d0941964aa3ebdcb00ccef58b1bb399f9f898465e9886d5aec7f31090a0fb30\""}})

	am.AddFS(sampleFiles)
	tR(t, am, "/assets/internal/asset.sample", 200, []byte("Sample Asset"), nil, http.Header{"Etag": {"\"682371dd5ee6536924b1ae1e477e65baf4c67e3a060e7a08d561e5981247e234\""}})

}
