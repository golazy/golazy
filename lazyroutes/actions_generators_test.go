package lazyroutes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type actionArgumentController struct{}

func newActionArgumentController(context.Context) (*actionArgumentController, error) {
	return &actionArgumentController{}, nil
}

func (c *actionArgumentController) Show(w http.ResponseWriter, postID int, slug string) error {
	_, err := fmt.Fprintf(w, "post:%d slug:%s", postID, slug)
	return err
}

func TestControllerActionResolvesRouteParameters(t *testing.T) {
	scope := New(context.Background())
	scope.Get(
		"/posts/{post_id}",
		newActionArgumentController,
		(*actionArgumentController).Show,
	)

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/posts/42", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "post:42 slug:42"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

type generatedInput struct {
	Title string
}

type generatorController struct{}

func newGeneratorController(context.Context) (*generatorController, error) {
	return &generatorController{}, nil
}

func (c *generatorController) GenPostInput(r *http.Request) (generatedInput, error) {
	title := r.URL.Query().Get("title")
	if title == "" {
		return generatedInput{}, errors.New("title is required")
	}
	return generatedInput{Title: title}, nil
}

func (c *generatorController) Create(w http.ResponseWriter, input generatedInput) error {
	_, err := fmt.Fprintf(w, "created:%s", input.Title)
	return err
}

func (c *generatorController) HandleError(w http.ResponseWriter, _ *http.Request, err error) error {
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, writeErr := fmt.Fprint(w, err.Error())
	return writeErr
}

func TestControllerActionUsesGenerator(t *testing.T) {
	scope := New(context.Background())
	scope.Post("/posts", newGeneratorController, (*generatorController).Create)

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/posts?title=hello", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "created:hello"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestControllerGeneratorErrorUsesControllerErrorHandler(t *testing.T) {
	scope := New(context.Background())
	scope.Post("/posts", newGeneratorController, (*generatorController).Create)

	response := httptest.NewRecorder()
	scope.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/posts", nil))

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnprocessableEntity)
	}
	if got, want := response.Body.String(), "title is required"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

type currentUser struct {
	Name string
}

type generatedPost struct {
	ID   int
	User string
}

type generatorChainController struct {
	userCalls int
	postCalls int
}

func newGeneratorChainController(context.Context) (*generatorChainController, error) {
	return &generatorChainController{}, nil
}

func (c *generatorChainController) GenCurrentUser(r *http.Request) currentUser {
	c.userCalls++
	return currentUser{Name: r.Header.Get("X-User")}
}

func (c *generatorChainController) GenGeneratedPost(id int, user currentUser) (generatedPost, error) {
	c.postCalls++
	return generatedPost{ID: id, User: user.Name}, nil
}

func (c *generatorChainController) Show(w http.ResponseWriter, post generatedPost, user currentUser) error {
	_, err := fmt.Fprintf(
		w,
		"post:%d user:%s direct:%s userCalls:%d postCalls:%d",
		post.ID,
		post.User,
		user.Name,
		c.userCalls,
		c.postCalls,
	)
	return err
}

func TestControllerGeneratorsCanRequireOtherGenerators(t *testing.T) {
	scope := New(context.Background())
	scope.Get("/posts/{post_id}", newGeneratorChainController, (*generatorChainController).Show)

	request := httptest.NewRequest(http.MethodGet, "/posts/42", nil)
	request.Header.Set("X-User", "g")
	response := httptest.NewRecorder()
	scope.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	want := "post:42 user:g direct:g userCalls:1 postCalls:1"
	if got := response.Body.String(); got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

type cycleA struct{}
type cycleB struct{}

type generatorCycleController struct{}

func newGeneratorCycleController(context.Context) (*generatorCycleController, error) {
	return &generatorCycleController{}, nil
}

func (c *generatorCycleController) GenCycleA(cycleB) (cycleA, error) {
	return cycleA{}, nil
}

func (c *generatorCycleController) GenCycleB(cycleA) (cycleB, error) {
	return cycleB{}, nil
}

func (c *generatorCycleController) Index(cycleA) error {
	return nil
}

func TestControllerGeneratorCycleFailsAtRouteRegistration(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if message := fmt.Sprint(recovered); !strings.Contains(message, "generator cycle") {
			t.Fatalf("panic = %q, want generator cycle", message)
		}
	}()

	scope := New(context.Background())
	scope.Get("/cycle", newGeneratorCycleController, (*generatorCycleController).Index)
}
