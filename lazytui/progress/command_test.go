package progress

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestCmdUsesExplicitArgs(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	withRunProgram(t, func(_ io.Reader, stdout io.Writer, stderr io.Writer, command string, args []string) error {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		_, _ = stdout.Write([]byte("ok"))
		_, _ = stderr.Write([]byte("diagnostic"))
		return nil
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Cmd("go", "test", "./...")(strings.NewReader(""), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if gotCommand != "go" {
		t.Fatalf("command = %q, want go", gotCommand)
	}
	if want := []string{"test", "./..."}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
	if stdout.String() != "ok" {
		t.Fatalf("stdout = %q, want ok", stdout.String())
	}
	if stderr.String() != "diagnostic" {
		t.Fatalf("stderr = %q, want diagnostic", stderr.String())
	}
}

func TestCmdSplitsCommandString(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, command string, args []string) error {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		return nil
	})

	err := Cmd(`go test "./with space" escaped\ value`)(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotCommand != "go" {
		t.Fatalf("command = %q, want go", gotCommand)
	}
	if want := []string{"test", "./with space", "escaped value"}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCmdReturnsSplitErrorWhenFunctionRuns(t *testing.T) {
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, _ string, _ []string) error {
		t.Fatal("runProgram should not be called after split error")
		return nil
	})

	err := Cmd(`go "unterminated`)(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err == nil {
		t.Fatal("expected split error")
	}
	if !strings.Contains(err.Error(), "unterminated quote") {
		t.Fatalf("error = %v, want unterminated quote", err)
	}
}

func TestCmdWarnWrapsFailure(t *testing.T) {
	failure := errors.New("failed")
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, _ string, _ []string) error {
		return failure
	})

	err := CmdWarn("go", "test")(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err == nil {
		t.Fatal("expected warning")
	}
	var warn Warn
	if !errors.As(err, &warn) {
		t.Fatalf("error = %T, want Warn", err)
	}
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want wrapped failure", err)
	}
}

func TestMiseBuildsRawExecCommand(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, command string, args []string) error {
		gotCommand = command
		gotArgs = append([]string(nil), args...)
		return nil
	})

	err := Mise("go", "test", "./...")(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotCommand != "mise" {
		t.Fatalf("command = %q, want mise", gotCommand)
	}
	if want := []string{"exec", "--raw", "--", "go", "test", "./..."}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestMiseSplitsCommandString(t *testing.T) {
	var gotArgs []string
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, _ string, args []string) error {
		gotArgs = append([]string(nil), args...)
		return nil
	})

	err := Mise(`go test ./...`)(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"exec", "--raw", "--", "go", "test", "./..."}; !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestMiseWarnWrapsFailure(t *testing.T) {
	failure := errors.New("failed")
	withRunProgram(t, func(_ io.Reader, _ io.Writer, _ io.Writer, _ string, _ []string) error {
		return failure
	})

	err := MiseWarn("go test")(
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	)
	if err == nil {
		t.Fatal("expected warning")
	}
	var warn Warn
	if !errors.As(err, &warn) {
		t.Fatalf("error = %T, want Warn", err)
	}
	if !errors.Is(err, failure) {
		t.Fatalf("error = %v, want wrapped failure", err)
	}
}

func withRunProgram(t *testing.T, fn func(io.Reader, io.Writer, io.Writer, string, []string) error) {
	t.Helper()
	original := runProgram
	runProgram = fn
	t.Cleanup(func() {
		runProgram = original
	})
}
