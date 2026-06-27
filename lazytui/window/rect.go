package window

import (
	"errors"

	"golazy.dev/lazytui/encoding/tty"
)

var ErrInvalidGeometry = errors.New("invalid terminal window geometry")

// Rect is a 1-based terminal-cell rectangle.
type Rect struct {
	Row  int
	Col  int
	Rows int
	Cols int
}

// Size returns the rectangle's terminal size.
func (r Rect) Size() tty.Size {
	return tty.Size{Rows: r.Rows, Cols: r.Cols}
}

// SplitSideBySide returns two pane rectangles inside total, after subtracting
// horizontal and vertical padding from the outside edges.
func SplitSideBySide(total tty.Size, paddingCols int, paddingRows int) ([2]Rect, error) {
	if !total.Valid() || paddingCols < 0 || paddingRows < 0 {
		return [2]Rect{}, ErrInvalidGeometry
	}

	rows := total.Rows - paddingRows*2
	cols := total.Cols - paddingCols*2
	if rows <= 0 || cols < 2 {
		return [2]Rect{}, ErrInvalidGeometry
	}

	leftCols := cols / 2
	rightCols := cols - leftCols
	if leftCols <= 0 || rightCols <= 0 {
		return [2]Rect{}, ErrInvalidGeometry
	}

	row := paddingRows + 1
	leftCol := paddingCols + 1
	rightCol := leftCol + leftCols
	return [2]Rect{
		{Row: row, Col: leftCol, Rows: rows, Cols: leftCols},
		{Row: row, Col: rightCol, Rows: rows, Cols: rightCols},
	}, nil
}
