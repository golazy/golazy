package multihttp

import (
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golazy/golazy/autocerts"

	"github.com/adrg/xdg"
	"github.com/quic-go/quic-go/http3"
)

func getTLSConfig() *tls.Config {
	file, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		file = "golazy.pem"
	}

	tls, err := autocerts.TLSConfigFile(file)
	if err != nil {
		panic(err)
	}
	return tls
}

func TestServe(t *testing.T) {

	s := &Server{
		Addr: "127.0.0.1:1999",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hi"))
		}),
		TLSConfig: getTLSConfig(),
	}

	done := make(chan (struct{}))
	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Error(err)
		}
		done <- struct{}{}
	}()

	time.Sleep(1 * time.Second)

	// http
	res, err := http.Get("http://127.0.0.1:1999")
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if string(body) != "hi" {
		t.Fatal("expected hi, got", string(body))
	}

	// https
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err = http.Get("https://127.0.0.1:1999")
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if string(body) != "hi" {
		t.Fatal("expected hi, got", string(body))
	}

	// http3
	c := &http.Client{
		Transport: &http3.RoundTripper{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	res, err = c.Get("https://127.0.0.1:1999")
	if err != nil {
		t.Error(err)
	}
	defer res.Body.Close()

	body, err = io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
	if string(body) != "hi" {
		t.Fatal("expected hi, got", string(body))
	}

	err = s.Close()
	if err != nil {
		t.Error(err)
	}

	<-done

}

/*
func TestProtocols(t *testing.T) {

	s := &Server{
		Addr: ":9000",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			for {
				select {
				case <-r.Context().Done():
					return
				case <-time.After(1 * time.Second):
					fmt.Fprintf(w, "data: %s\n\n", "hello")
					w.(http.Flusher).Flush()
				}
			}

		}),
		TLSConfig: getTLSConfig(),
	}

	s.ListenAndServe()
}

*/
