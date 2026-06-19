package actioncall

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type contextValueKey struct{}

type defaultArgumentController struct {
	ctx        context.Context
	lastString string
	allStrings []string
	lastInt    int
	allInts    []int
}

func (c *defaultArgumentController) Capture(ctx context.Context, lastString string, allStrings []string, lastInt int, allInts []int) error {
	c.ctx = ctx
	c.lastString = lastString
	c.allStrings = allStrings
	c.lastInt = lastInt
	c.allInts = allInts
	return nil
}

func TestDefaultGeneratedArgumentsResolvePathVariables(t *testing.T) {
	tests := []struct {
		name       string
		routePath  string
		pathValues map[string]string
		lastString string
		allStrings []string
		lastInt    int
		allInts    []int
	}{
		{
			name:      "uses last and all path variables",
			routePath: "/teams/{team_id}/posts/{post_id}",
			pathValues: map[string]string{
				"team_id": "17",
				"post_id": "42",
			},
			lastString: "42",
			allStrings: []string{"17", "42"},
			lastInt:    42,
			allInts:    []int{17, 42},
		},
		{
			name:      "failed int conversions become zero",
			routePath: "/teams/{team_id}/posts/{post_id}",
			pathValues: map[string]string{
				"team_id": "17",
				"post_id": "draft",
			},
			lastString: "draft",
			allStrings: []string{"17", "draft"},
			lastInt:    0,
			allInts:    []int{17, 0},
		},
		{
			name:       "routes without variables use zero values",
			routePath:  "/posts",
			lastString: "",
			allStrings: []string{},
			lastInt:    0,
			allInts:    []int{},
		},
	}

	action := reflect.ValueOf((*defaultArgumentController).Capture)
	controllerType := reflect.TypeOf((*defaultArgumentController)(nil))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := Compile(controllerType, action, Options{RoutePath: tt.routePath})
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request = request.WithContext(context.WithValue(request.Context(), contextValueKey{}, "request-context"))
			for name, value := range tt.pathValues {
				request.SetPathValue(name, value)
			}

			controller := &defaultArgumentController{}
			err = plan.Call(reflect.ValueOf(controller), httptest.NewRecorder(), request)
			if err != nil {
				t.Fatalf("Call() error = %v", err)
			}

			if got := controller.ctx.Value(contextValueKey{}); got != "request-context" {
				t.Fatalf("context value = %v, want request-context", got)
			}
			if controller.lastString != tt.lastString {
				t.Fatalf("lastString = %q, want %q", controller.lastString, tt.lastString)
			}
			if !reflect.DeepEqual(controller.allStrings, tt.allStrings) {
				t.Fatalf("allStrings = %#v, want %#v", controller.allStrings, tt.allStrings)
			}
			if controller.lastInt != tt.lastInt {
				t.Fatalf("lastInt = %d, want %d", controller.lastInt, tt.lastInt)
			}
			if !reflect.DeepEqual(controller.allInts, tt.allInts) {
				t.Fatalf("allInts = %#v, want %#v", controller.allInts, tt.allInts)
			}
		})
	}
}

type generatedArgument struct {
	ID         int
	ContextTag string
}

type generatorArgumentController struct {
	arg generatedArgument
}

func (c *generatorArgumentController) GenGeneratedArgument(id int, ctx context.Context) generatedArgument {
	return generatedArgument{
		ID:         id,
		ContextTag: ctx.Value(contextValueKey{}).(string),
	}
}

func (c *generatorArgumentController) Show(arg generatedArgument) error {
	c.arg = arg
	return nil
}

func TestCustomGeneratedTypesCanUseDefaultArguments(t *testing.T) {
	plan, err := Compile(
		reflect.TypeOf((*generatorArgumentController)(nil)),
		reflect.ValueOf((*generatorArgumentController).Show),
		Options{RoutePath: "/posts/{post_id}"},
	)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(context.WithValue(request.Context(), contextValueKey{}, "generator-context"))
	request.SetPathValue("post_id", "42")

	controller := &generatorArgumentController{}
	err = plan.Call(reflect.ValueOf(controller), httptest.NewRecorder(), request)
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	want := generatedArgument{ID: 42, ContextTag: "generator-context"}
	if controller.arg != want {
		t.Fatalf("generated argument = %#v, want %#v", controller.arg, want)
	}
}
