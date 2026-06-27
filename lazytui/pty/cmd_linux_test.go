//go:build linux

package pty

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"

	"golazy.dev/lazytui/encoding/tty"
)

func TestCommandReportsInitialSize(t *testing.T) {
	var out bytes.Buffer
	cmd := shellCommand(t, "stty size")
	cmd.Size = tty.Size{Rows: 7, Cols: 19}
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	requireFields(t, out.String(), []string{"7", "19"})
}

func TestCommandResize(t *testing.T) {
	var out bytes.Buffer
	cmd := shellCommand(t, "stty size; sleep 0.2; stty size")
	cmd.Size = tty.Size{Rows: 5, Cols: 11}
	cmd.Stdout = &out

	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := cmd.Resize(tty.Size{Rows: 6, Cols: 12}); err != nil {
		t.Fatalf("Resize returned error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	requireFields(t, out.String(), []string{"5", "11", "6", "12"})
}

func TestCommandCopiesInputAndExposesHandles(t *testing.T) {
	var out bytes.Buffer
	cmd := shellCommand(t, "read line; printf '<%s>' \"$line\"")
	cmd.Size = tty.Size{Rows: 4, Cols: 20}
	cmd.Stdin = strings.NewReader("lazy\n")
	cmd.Stdout = &out

	if cmd.Master() != nil {
		t.Fatal("Master returned file before Start")
	}
	if cmd.Process() != nil {
		t.Fatal("Process returned process before Start")
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if cmd.Master() == nil {
		t.Fatal("Master returned nil after Start")
	}
	if cmd.Process() == nil {
		t.Fatal("Process returned nil after Start")
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
	if !strings.Contains(out.String(), "<lazy>") {
		t.Fatalf("output = %q, want copied input", out.String())
	}
}

func TestCommandRejectsSecondStart(t *testing.T) {
	cmd := shellCommand(t, "exit 0")
	if err := cmd.Start(); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if err := cmd.Start(); err != ErrAlreadyStarted {
		t.Fatalf("second Start error = %v, want ErrAlreadyStarted", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("Wait returned error: %v", err)
	}
}

func shellCommand(t *testing.T, script string) *Cmd {
	t.Helper()

	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("sh not available")
	}
	return Command(shell, "-c", script)
}

func requireFields(t *testing.T, output string, want []string) {
	t.Helper()

	got := strings.Fields(output)
	if len(got) != len(want) {
		t.Fatalf("fields = %#v, want %#v; output %q", got, want, output)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("fields = %#v, want %#v; output %q", got, want, output)
		}
	}
}
