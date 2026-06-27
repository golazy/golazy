package pty

import (
	"errors"
	"testing"

	"golazy.dev/lazytui/encoding/tty"
)

func TestCommandRejectsInvalidInput(t *testing.T) {
	if err := Command("").Start(); !errors.Is(err, ErrInvalidCommand) {
		t.Fatalf("Start error = %v, want ErrInvalidCommand", err)
	}

	cmd := Command("sh")
	cmd.Size = tty.Size{}
	if err := cmd.Start(); !errors.Is(err, tty.ErrInvalidSize) {
		t.Fatalf("Start error = %v, want ErrInvalidSize", err)
	}

	if err := Command("sh").Wait(); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Wait error = %v, want ErrNotStarted", err)
	}
	if err := Command("sh").Resize(tty.Size{Rows: 1, Cols: 1}); !errors.Is(err, ErrNotStarted) {
		t.Fatalf("Resize error = %v, want ErrNotStarted", err)
	}
	if err := Command("sh").Resize(tty.Size{}); !errors.Is(err, tty.ErrInvalidSize) {
		t.Fatalf("Resize error = %v, want ErrInvalidSize", err)
	}
}
