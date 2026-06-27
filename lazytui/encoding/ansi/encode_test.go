package ansi

import (
	"bytes"
	"testing"

	"golazy.dev/lazytui/encoding/ansi/codes"
)

func TestAppendOperations(t *testing.T) {
	tests := map[string]struct {
		op   Op
		want string
	}{
		"print":        {Print("hello"), "hello"},
		"control":      {Control(codes.CR), "\r"},
		"escape":       {Escape('7'), "\x1b7"},
		"csi":          {CSI(codes.CUP, 12, 4), "\x1b[12;4H"},
		"prefixed csi": {CSIWithPrefix("?", codes.SM, codes.MouseSGR), "\x1b[?1006h"},
		"movement":     {CursorPosition(3, 7), "\x1b[3;7H"},
		"size request": {RequestWindowSize(), "\x1b[18t"},
		"size report":  {WindowSizeReport(24, 80), "\x1b[8;24;80t"},
		"mouse enable": {EnableMouse(), "\x1b[?1002h\x1b[?1006h"},
		"mouse disable": {
			DisableMouse(),
			"\x1b[?1006l\x1b[?1003l\x1b[?1002l\x1b[?1000l\x1b[?9l",
		},
		"sgr":     {SGR(codes.Bold, codes.FgRGB(120, 80, 220)), "\x1b[1;38;2;120;80;220m"},
		"title":   {WindowTitle("Lazy"), "\x1b]2;Lazy\x1b\\"},
		"url":     {Hyperlink("https://golazy.dev"), "\x1b]8;;https://golazy.dev\x1b\\"},
		"end url": {EndHyperlink(), "\x1b]8;;\x1b\\"},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := string(Append(nil, test.op)); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestEncoderWritesOperation(t *testing.T) {
	var buf bytes.Buffer
	encoder := &Encoder{}

	if err := encoder.Encode(&buf, CSI(codes.EL, 2)); err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if got, want := buf.String(), "\x1b[2K"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestAppendNilOperation(t *testing.T) {
	got := string(Append([]byte("prefix"), nil))
	if got != "prefix" {
		t.Fatalf("got %q, want prefix", got)
	}
}
