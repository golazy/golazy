package ansi

import "testing"

func TestTerminalModeOperations(t *testing.T) {
	tests := []struct {
		name string
		op   Op
		want string
	}{
		{"alternate on", EnterAlternateBuffer(), "\x1b[?1049h"},
		{"alternate off", LeaveAlternateBuffer(), "\x1b[?1049l"},
		{"cursor hide", HideCursor(), "\x1b[?25l"},
		{"cursor show", ShowCursor(), "\x1b[?25h"},
		{"focus on", EnableFocus(), "\x1b[?1004h"},
		{"focus off", DisableFocus(), "\x1b[?1004l"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := string(Append(nil, test.op)); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
