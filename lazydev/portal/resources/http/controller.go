package http

import (
	"bytes"
	"encoding/pem"
	"io"
	"net/http"
	"os"
	"portal/assets"
	"portal/layouts/golazy"

	_ "embed"

	"github.com/adrg/xdg"
	"golazy.dev/lazyaction"
	"golazy.dev/lazyview/script"
)

//go:embed redirect.js
var redirectJS []byte

const redirectPath = "/golazy/http/redirect.js"

func init() {
	assets.Assets.AddFile(redirectPath, redirectJS)
}

type Controller struct {
	golazy.Layout
}

func (h *Controller) Index(r *http.Request) string {

	h.AddScript(script.Script{
		Src:            "https://" + r.Host + "/golazy/http/redirect",
		Priority:       script.High,
		Referrerpolicy: script.NoReferrer,
		Async:          true,
		CrossOrigin:    script.Anonymous,
	})

	return "hola"

}

func (h *Controller) GetRedirect(w http.ResponseWriter) {
	h.SkipLayout()
	w.Header().Add("Content-Type", "application/javascript")
	// Add cors headers to allow any origin
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Write(redirectJS)
}

func (a *Controller) GetDownloadCert(c *lazyaction.Context) {
	file, err := xdg.DataFile("golazy/golazy.pem")
	if err != nil {
		return
	}
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()

	cert, err := io.ReadAll(f)
	if err != nil {
		return
	}

	var block *pem.Block
	for {
		block, cert = pem.Decode(cert)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}

		data := &bytes.Buffer{}
		pem.Encode(data, block)

		c.SendFile("golazy.pem", data)
		return
	}

}
