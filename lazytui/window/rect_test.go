package window

import (
	"errors"
	"testing"

	"golazy.dev/lazytui/encoding/tty"
)

func TestSplitSideBySide(t *testing.T) {
	panes, err := SplitSideBySide(tty.Size{Rows: 5, Cols: 13}, 2, 1)
	if err != nil {
		t.Fatalf("SplitSideBySide returned error: %v", err)
	}

	want := [2]Rect{
		{Row: 2, Col: 3, Rows: 3, Cols: 4},
		{Row: 2, Col: 7, Rows: 3, Cols: 5},
	}
	if panes != want {
		t.Fatalf("panes = %#v, want %#v", panes, want)
	}
	if size := panes[0].Size(); size != (tty.Size{Rows: 3, Cols: 4}) {
		t.Fatalf("left size = %v", size)
	}
}

func TestSplitSideBySideRejectsInvalidGeometry(t *testing.T) {
	tests := map[string]struct {
		size        tty.Size
		paddingCols int
		paddingRows int
	}{
		"empty size":     {},
		"negative col":   {tty.Size{Rows: 4, Cols: 10}, -1, 0},
		"negative row":   {tty.Size{Rows: 4, Cols: 10}, 0, -1},
		"no inner rows":  {tty.Size{Rows: 2, Cols: 10}, 0, 1},
		"one inner col":  {tty.Size{Rows: 4, Cols: 5}, 2, 0},
		"zero inner col": {tty.Size{Rows: 4, Cols: 4}, 2, 0},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := SplitSideBySide(test.size, test.paddingCols, test.paddingRows)
			if !errors.Is(err, ErrInvalidGeometry) {
				t.Fatalf("error = %v, want ErrInvalidGeometry", err)
			}
		})
	}
}
