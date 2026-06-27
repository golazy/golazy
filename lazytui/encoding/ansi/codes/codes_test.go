package codes

import "testing"

func TestControlValues(t *testing.T) {
	tests := map[string]struct {
		got  Control
		want byte
	}{
		"BEL": {BEL, 0x07},
		"ESC": {ESC, 0x1b},
		"DEL": {DEL, 0x7f},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if byte(test.got) != test.want {
				t.Fatalf("got %#x, want %#x", byte(test.got), test.want)
			}
		})
	}
}

func TestCSIFinalValues(t *testing.T) {
	tests := map[string]struct {
		got  byte
		want byte
	}{
		"CUP": {CUP, 'H'},
		"ED":  {ED, 'J'},
		"SM":  {SM, 'h'},
		"RM":  {RM, 'l'},
		"SGR": {SGRFinal, 'm'},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestSGRHelpers(t *testing.T) {
	tests := map[string]struct {
		got  SGR
		want []int
	}{
		"foreground 256": {Fg256(12), []int{38, 5, 12}},
		"background 256": {Bg256(240), []int{48, 5, 240}},
		"foreground rgb": {FgRGB(1, 2, 3), []int{38, 2, 1, 2, 3}},
		"background rgb": {BgRGB(4, 5, 6), []int{48, 2, 4, 5, 6}},
		"overline":       {Overline, []int{53}},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if len(test.got) != len(test.want) {
				t.Fatalf("got len %d, want %d", len(test.got), len(test.want))
			}
			for i := range test.want {
				if test.got[i] != test.want[i] {
					t.Fatalf("at %d got %d, want %d", i, test.got[i], test.want[i])
				}
			}
		})
	}
}
