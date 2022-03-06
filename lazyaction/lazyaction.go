package lazyaction

import (
	"fmt"
	"io"
	"net/http"
	"reflect"
	"runtime"

	. "github.com/guillermo/golazy/lazyview/html"
	"github.com/guillermo/golazy/lazyview/layout"
	"github.com/guillermo/golazy/lazyview/layout/lazylayout"
)

type Controller struct {
	path     string
	dest     interface{}
	template *layout.LayoutTemplate
}

func (c *Controller) error(w http.ResponseWriter, r *http.Request, code int, err error) {
	w.WriteHeader(code)

	lazylayout.Layout.With(
		lazylayout.PageHeader(),
		lazylayout.PageNav(),
		Main(H1("Error"),
			Pre(err.Error())),
	).WriteTo(w)

}

func (c *Controller) restIndex(w http.ResponseWriter, r *http.Request) interface{} {
	v, ok := c.dest.(interface {
		Index(w http.ResponseWriter, r *http.Request) interface{}
	})
	if !ok {
		c.error(w, r, http.StatusMethodNotAllowed, fmt.Errorf("the resource does not implement that method"))
		return nil
	}
	return v.Index(w, r)
}

func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		ret := c.restIndex(w, r)
		switch content := ret.(type) {
		case nil:
			return
		case io.WriterTo:
			layouter, ok := c.dest.(interface {
				Layout(*http.Request) *layout.LayoutTemplate
			})
			if ok {
				l := layouter.Layout(r)
				l.With(content).WriteTo(w)
				return
			} else {
				content.WriteTo(w)
				return
			}
		case error:
			c.error(w, r, 500, content)
			return
		default:
			c.error(w, r, 500, fmt.Errorf("the action returned an unknonwn value %+v", content))
		}
	}

	http.Error(w, "The resource does not implement that method", http.StatusMethodNotAllowed)
}

func applicationController(path string, dest interface{}) http.Handler {
	return &Controller{
		path: path,
		dest: dest,
		template: &layout.LayoutTemplate{
			Title: runtime.FuncForPC(reflect.ValueOf(dest).Pointer()).Name(),
		},
	}

}

func Route(path string, dest interface{}) {
	http.Handle(path, applicationController(path, dest))
}
