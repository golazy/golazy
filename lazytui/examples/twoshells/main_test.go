package main

import (
	"bytes"
	"errors"
	"os/exec"
	"testing"

	"golazy.dev/lazytui/window"
)

func TestRunProbeSpawnsTwoSizedShells(t *testing.T) {
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run([]string{
		"-probe",
		"-shell", shell,
		"-rows", "3",
		"-cols", "12",
		"-pad-x", "2",
		"-pad-y", "1",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, stderr.String())
	}

	if got, want := stdout.String(), "left 1 4\nright 1 4\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestRunRejectsInvalidGeometry(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := run([]string{"-probe", "-rows", "1", "-cols", "4", "-pad-y", "1"}, &stdout, &stderr)
	if !errors.Is(err, window.ErrInvalidGeometry) {
		t.Fatalf("run error = %v, want ErrInvalidGeometry", err)
	}
}

func TestResolveShellExplicit(t *testing.T) {
	shell, err := resolveShell("custom-shell")
	if err != nil {
		t.Fatalf("resolveShell returned error: %v", err)
	}
	if shell != "custom-shell" {
		t.Fatalf("shell = %q, want custom-shell", shell)
	}
}
