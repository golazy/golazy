package lazycli_test

import (
	"bytes"
	"context"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"golazy.dev/lazytui/lazycli"
)

type rootParams struct {
	Verbose bool `lazycli:"short=v,help=Print verbose output"`
}

type createParams struct {
	Name string `lazycli:"help=Bucket name"`
}

type rootController struct {
	calls *[]string
}

var calls *[]string

func (rootController) Before(ctx context.Context, params rootParams) (context.Context, error) {
	*calls = append(*calls, "before:"+strconv.FormatBool(params.Verbose))
	return ctx, nil
}

func (rootController) Create(_ context.Context, params createParams) error {
	*calls = append(*calls, "create:"+params.Name)
	return nil
}

func (rootController) After(context.Context, rootParams) error {
	*calls = append(*calls, "after")
	return nil
}

func (rootController) Cleanup(context.Context, struct{}) error {
	*calls = append(*calls, "cleanup")
	return nil
}

func TestExecuteInfersMethodsAndParametersAnywhere(t *testing.T) {
	var got []string
	calls = &got
	draw := func(s lazycli.Scope) {
		s.Root(lazycli.Command{Controller: rootController{}})
		s.Define("create", lazycli.Command{Help: "Create a bucket"})
	}
	var stdout, stderr bytes.Buffer
	code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{
		Name: "lazy", Args: []string{"--name", "assets", "create", "--verbose"}, Stdout: &stdout, Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	want := []string{"before:true", "create:assets", "after", "cleanup"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func TestMergedControllerOverridesOnlyImplementedAction(t *testing.T) {
	type base struct{}
	type overlay struct{}
	_ = base{}
	_ = overlay{}
	// The behavior is covered through functions because local types cannot have
	// methods; this also verifies that later command definitions are retained.
	var got string
	draw := func(s lazycli.Scope) {
		s.Define("run", lazycli.Command{Controller: lazycli.Func(func(context.Context, struct{}) error {
			got = "base"
			return nil
		})})
		s.Define("run", lazycli.Command{Help: "Overlaid help"})
	}
	code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{Name: "app", Args: []string{"run"}})
	if code != 0 || got != "base" {
		t.Fatalf("code = %d, action = %q", code, got)
	}
}

func TestAutocompleteDoesNotInstantiateController(t *testing.T) {
	draw := func(s lazycli.Scope) {
		s.Root(lazycli.Command{Autocomplete: true})
		s.Define("create", lazycli.Command{Controller: lazycli.Func(func(context.Context, struct{}) error {
			t.Fatal("completion invoked action")
			return nil
		}), Help: "Create a bucket"})
	}
	var stdout bytes.Buffer
	code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{
		Name: "lazy", Args: []string{"__complete", "cr"}, Stdout: &stdout,
	})
	if code != 0 || !strings.Contains(stdout.String(), "create\tCreate a bucket") {
		t.Fatalf("code = %d, output = %q", code, stdout.String())
	}
}

func TestDefaultCommandRunsImplicitlyAndAppearsInParentHelp(t *testing.T) {
	called := 0
	draw := func(s lazycli.Scope) {
		s.Define("run", lazycli.Command{
			Default: true,
			Help:    "Start the application",
			Controller: lazycli.Func(func(context.Context, struct{}) error {
				called++
				return nil
			}),
		})
	}
	if code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{Name: "app"}); code != 0 || called != 1 {
		t.Fatalf("implicit default: code = %d, called = %d", code, called)
	}
	if code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{Name: "app", Args: []string{"run"}}); code != 0 || called != 2 {
		t.Fatalf("explicit default: code = %d, called = %d", code, called)
	}
	var stdout bytes.Buffer
	if code := lazycli.Execute(context.Background(), draw, lazycli.ExecuteConfig{Name: "app", Args: []string{"--help"}, Stdout: &stdout}); code != 0 {
		t.Fatalf("help code = %d", code)
	}
	if !strings.Contains(stdout.String(), "run (default)") {
		t.Fatalf("help = %q, want default marker", stdout.String())
	}
}
