package lazyapp

import (
	"bytes"
	"context"
	"io/fs"
	"testing"

	"golazy.dev/lazytui/lazycli"
)

func TestCLIHelpDoesNotInitializeApplication(t *testing.T) {
	config := Config{
		Public: func() (fs.FS, error) {
			t.Fatal("help initialized application files")
			return nil, nil
		},
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), config, lazycli.ExecuteConfig{
		Name: "example", Args: []string{"--help"}, Stdout: &stdout, Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("routes")) || !bytes.Contains(stdout.Bytes(), []byte("assets")) {
		t.Fatalf("help = %q, want framework commands", stdout.String())
	}
}

func TestApplicationRootRunOverridesDefaultServerAction(t *testing.T) {
	called := false
	config := Config{
		CLI: func(s lazycli.Scope) {
			s.Root(lazycli.Command{Controller: lazycli.Func(func(context.Context, struct{}) error {
				called = true
				return nil
			})})
		},
	}
	var stdout, stderr bytes.Buffer
	code := Execute(context.Background(), config, lazycli.ExecuteConfig{
		Name: "example", Stdout: &stdout, Stderr: &stderr,
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %s", code, stderr.String())
	}
	if !called {
		t.Fatal("application root action was not called")
	}
}
