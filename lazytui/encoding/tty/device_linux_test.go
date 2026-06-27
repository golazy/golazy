//go:build linux

package tty

import (
	"errors"
	"os"
	"testing"
)

func TestOpenNonTerminalFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "not-a-terminal")
	if err != nil {
		t.Fatalf("CreateTemp returned error: %v", err)
	}
	defer file.Close()

	device, err := Open(file)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	if device.IsTerminal() {
		t.Fatal("regular file reported terminal")
	}
	if _, err := device.Size(); err == nil {
		t.Fatal("expected Size to fail for regular file")
	}
	if err := device.Resize(Size{}); !errors.Is(err, ErrInvalidSize) {
		t.Fatalf("Resize invalid error = %v, want ErrInvalidSize", err)
	}
	if err := device.Resize(Size{Rows: 1, Cols: 1}); err == nil {
		t.Fatal("expected Resize to fail for regular file")
	}
	if _, err := device.MakeRaw(); err == nil {
		t.Fatal("expected MakeRaw to fail for regular file")
	}
	if err := device.Restore(newState("not-termios")); !errors.Is(err, ErrUnknownState) {
		t.Fatalf("Restore error = %v, want ErrUnknownState", err)
	}
}
