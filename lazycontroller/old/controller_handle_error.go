package lazycontroller

import (
	"errors"
	"fmt"
	"net/http"

	"golazy.dev/lazysupport"

	"golazy.dev/lazyml/html"
)

var FlashNotice = "notice"   // Blue
var FlashWarning = "warning" // Yellow
var FlashSuccess = "success" // Green
var FlashError = "error"     // Red

func (c *Controller) HandleError(err error) {
	fmt.Printf("Handling error : %+T\n", c.ResponseWriter)
	fmt.Println("    ↱ 🛑", cleanError(err))
	actionErr := &Error{}
	lazyErr := &lazysupport.ErrorWithStack{}

	handler := err
	for handler != nil {
		if err, ok := err.(http.Handler); ok {
			err.ServeHTTP(c.ResponseWriter, c.Request)
			return
		}
		if err, ok := err.(ErrorWithHandler); ok {
			err.Handler().ServeHTTP(c.ResponseWriter, c.Request)
			return
		}
		handler = errors.Unwrap(handler)
	}

	if err == ErrNotFound {
		c.ResponseWriter.WriteHeader(404)
		c.RenderContent(html.Main(
			html.Class("wrapper"),
			html.H1("Page not found", html.Class("h1 font-serif pt-8")),
		).String())
		return
	}

	switch {
	case errors.As(err, &actionErr):
		if actionErr.Location != "" {
			c.ResponseWriter.Header().Set("Location", actionErr.Location)
			if !actionErr.NoFlash {
				c.AddErrorFlash(actionErr)
			}
			code := actionErr.Code
			if code == 0 {
				code = 303
			}
			c.ResponseWriter.WriteHeader(code)
			fmt.Printf("%+T", c.ResponseWriter)
			c.ResponseWriter.Write([]byte(fmt.Sprintf("%+v", c.SessionValues())))
		} else {
			code := actionErr.Code
			if code == 0 {
				code = 500
			}
			c.ResponseWriter.WriteHeader(500)
			c.RenderContent(html.Main(
				html.Class("wrapper"),
				html.H1("Error", html.Class("h1 font-serif pt-8")),
				html.P(err.Error(), html.Class("text-red-500")),
			).String())
		}
		if actionErr.Raise {
			panic(err)
		}
	case errors.As(err, lazyErr):
		c.ResponseWriter.WriteHeader(500)
		c.RenderContent(html.Main(
			html.Class("wrapper"),
			html.H1("Error", html.Class("h1 font-serif pt-8")),
			html.P(err.Error(), html.Class("text-red-500")),
		).String())

	default:
		c.ResponseWriter.WriteHeader(500)
		c.RenderContent(html.Main(
			html.Class("wrapper"),
			html.H1("Error", html.Class("h1 font-serif pt-8")),
			html.P(err.Error(), html.Class("text-red-500")),
		).String())

	}

}
