package lazydispatch

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type BaseController struct {
	r   *http.Request
	w   http.ResponseWriter
	out string
}

func (c *BaseController) Before_000_SetRequestResponse(r *http.Request, w http.ResponseWriter) {
	c.r = r
	c.w = w
}

func (c *BaseController) After_ZZZ_WriteOutput() {
	if c.out != "" {
		c.w.Write([]byte(c.out))
	}
}

type MoviesController struct {
	BaseController
}
type PostParam string

func (c *MoviesController) Gen__PostParam() PostParam {
	return PostParam("index")
}

func (c *MoviesController) Index(p PostParam) {
	c.out = string(p)
}
func (c *MoviesController) Show(a, b string) string {
	return fmt.Sprintf("id1:%s id2:%s", a, b)
}

type ContextParam string

func (c *MoviesController) Get_Context(ctx context.Context) string {
	return ctx.Value("ctx_param").(string)
}

func (c *MoviesController) ParamFromContext(p ContextParam) {
	c.out = string(p)
}

func TestForAction_Context(t *testing.T) {
	expectForAction(t, forAction(&MoviesController{}, "Get_Context", func(ctx context.Context, r *http.Request) context.Context {
		return context.WithValue(ctx, "ctx_param", "context")
	}), "context", 200)

}

func TestForAction_Filters(t *testing.T) {
	expectForAction(t, forAction(&MoviesController{}, "Index"), "index", 200)
}

func ctxWithValue[T any](base context.Context, value T) context.Context {
	t := reflect.TypeOf(*new(T))
	return context.WithValue(base, t, value)
}

func TestForAction_ParamsFromContext(t *testing.T) {
	r := httptest.NewRequest("GET", "/posts", nil)
	r = r.WithContext(ctxWithValue(context.Background(), ContextParam("context")))
	w := httptest.NewRecorder()
	forAction(&MoviesController{}, "ParamFromContext").ServeHTTP(w, r)
	body := "context"
	code := 200

	if w.Body.String() != body {
		t.Errorf("expected %s, got %s", body, w.Body.String())
	}
	if w.Code != code {
		t.Errorf("expected %d, got %d", code, w.Code)
	}
}

func expectForAction(t *testing.T, handler http.Handler, body string, code int) {
	t.Helper()
	r := httptest.NewRequest("GET", "/posts", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Body.String() != body {
		t.Errorf("expected %s, got %s", body, w.Body.String())
	}
	if w.Code != code {
		t.Errorf("expected %d, got %d", code, w.Code)
	}

}

func TestForAction_StringParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/brands/33/posts/55", nil)
	w := httptest.NewRecorder()
	forAction(&MoviesController{}, "Show", func(ctx context.Context, r *http.Request) context.Context {
		return context.WithValue(ctx, reflect.TypeFor[*Route](), &Route{
			Path: "/brands/:brand_id/posts/:id",
		})
	}).ServeHTTP(w, r)
	body := "id1:55 id2:33"
	code := 200

	if w.Body.String() != body {
		t.Errorf("expected %s, got %s", body, w.Body.String())
	}
	if w.Code != code {
		t.Errorf("expected %d, got %d", code, w.Code)
	}
}

type GenErrorController struct {
	gen       error
	before    error
	after     error
	action    error
	personErr error
}

type Person struct{}

func (c *GenErrorController) HandleError(w http.ResponseWriter, err error) {
	w.WriteHeader(503)
	w.Write([]byte(err.Error()))
}

func (c *GenErrorController) Gen_User() (*User, error) {
	return nil, c.gen
}
func (c *GenErrorController) Gen_Person() (*Person, error) {
	return nil, c.personErr
}

func (c *GenErrorController) Before_Action(*Person) error {
	return c.before
}

func (c *GenErrorController) Index(w http.ResponseWriter, u *User) error {
	if c.gen != nil ||
		c.before != nil ||
		c.personErr != nil {
		panic("Index should not be called if there is an error before")
	}
	if c.action == nil {
		w.Write([]byte("ok"))
		return nil
	}
	return c.action
}

func (c *GenErrorController) After_Action() error {
	if c.before != nil ||
		c.action != nil ||
		c.gen != nil ||
		c.personErr != nil {
		panic("After should not be called if there is an error before")
	}
	return c.after
}

func TestForAction_Errors(t *testing.T) {
	err := fmt.Errorf("HandlerError")
	var tests = []struct {
		name    string
		handler http.Handler
		code    int
		body    string
	}{
		{"NoError", forAction(&GenErrorController{}, "Index"), 200, "ok"},
		{"GenError", forAction(&GenErrorController{gen: err}, "Index"), 503, "HandlerError"},
		{"BeforeError", forAction(&GenErrorController{before: err}, "Index"), 503, "HandlerError"},
		{"ActionError", forAction(&GenErrorController{action: err}, "Index"), 503, "HandlerError"},
		{"AfterError", forAction(&GenErrorController{after: err}, "Index"), 200, "okHandlerError"},
		{"GenOnFilterError", forAction(&GenErrorController{personErr: err}, "Index"), 503, "HandlerError"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/brands/33/posts/55", nil)
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			if w.Body.String() != tt.body {
				t.Errorf("expected %s, got %s", tt.body, w.Body.String())
			}
			if w.Code != tt.code {
				t.Errorf("expected %d, got %d", tt.code, w.Code)
			}
		})
	}

}

type PanicController struct {
	gen       error
	before    error
	action    error
	after     error
	personErr error
}

func (c *PanicController) HandleError(w http.ResponseWriter, err error) {
	w.WriteHeader(503)
	w.Write([]byte(err.Error()))
}

func (c *PanicController) Gen_User() *User {
	if c.gen != nil {
		panic(c.gen)
	}
	return nil
}
func (c *PanicController) Gen_Person() *Person {
	if c.personErr != nil {
		panic(c.personErr)
	}
	return nil
}

func (c *PanicController) Before_Action(*Person) {
	if c.before != nil {
		panic(c.before)
	}
}

func (c *PanicController) Index(w http.ResponseWriter, u *User) {
	if c.action != nil {
		panic(c.action)
	}

	w.Write([]byte("ok"))
}

func (c *PanicController) After_Action() {
	if c.after != nil {
		panic(c.after)
	}
}

func TestForAction_Panics(t *testing.T) {
	t.Skip("FIX")
	err := fmt.Errorf("pe")
	var tests = []struct {
		name    string
		handler http.Handler
		code    int
		body    string
	}{
		{"NoPanic", forAction(&PanicController{}, "Index"), 200, "ok"},
		{"GenPanic", forAction(&PanicController{gen: err}, "Index"), 503, "panic: pe"},
		{"BeforePanic", forAction(&PanicController{before: err}, "Index"), 503, "panic: pe"},
		{"ActionPanic", forAction(&PanicController{action: err}, "Index"), 503, "panic: pe"},
		{"AfterPanic", forAction(&PanicController{after: err}, "Index"), 200, "okpanic: pe"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/brands/33/posts/55", nil)
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			if w.Body.String() != tt.body {
				t.Errorf("expected %s, got %s", tt.body, w.Body.String())
			}
			if w.Code != tt.code {
				t.Errorf("expected %d, got %d", tt.code, w.Code)
			}
		})
	}

}

type Number int
type Counter struct {
	Error error
	Count Number
}

type OnceController struct {
	Counter Number
	Error   error
}

func (c *OnceController) Gen_Counter() (Number, error) {
	c.Counter++
	return c.Counter, c.Error
}

func (c *OnceController) Before_CountOne(i Number) {

}

func (c *OnceController) Index(i Number) string {
	return fmt.Sprint(i)
}

func TestGeneratorIsCalledOnlyOnce(t *testing.T) {

	var Tests = []struct {
		name    string
		out     string
		handler http.Handler
	}{
		//{"WithoutError", "2", ForAction(&OnceController{}, "Index")},
		//{"WithError", "1", ForAction(&OnceController{Error: fmt.Errorf("no no")}, "Index")},
	}

	for _, tt := range Tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/brands/33/posts/55", nil)
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			if w.Body.String() != tt.out {
				t.Errorf("expected %s. Got %s", tt.out, w.Body.String())
			}

		})

	}

}
