package lazytui

import (
	"fmt"
	"strconv"
	"strings"
)

// Color is a terminal color stored as an RGB value.
type Color struct {
	R uint8
	G uint8
	B uint8
}

// Hex parses a CSS-style six-digit hexadecimal color and panics when it is
// invalid. It is intended for package-level palette declarations.
func Hex(value string) Color {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		panic(fmt.Sprintf("lazytui: invalid color %q", value))
	}
	number, err := strconv.ParseUint(value, 16, 24)
	if err != nil {
		panic(fmt.Sprintf("lazytui: invalid color %q", value))
	}
	return Color{R: uint8(number >> 16), G: uint8(number >> 8), B: uint8(number)}
}

func (color Color) IsZero() bool {
	return color == Color{}
}

// Palette contains semantic colors shared by terminal-facing GoLazy packages.
// Text and terminal background colors are deliberately inherited from the
// terminal.
type Palette struct {
	Primary   Color
	Secondary Color
	Accent    Color
	Success   Color
	Warning   Color
	Error     Color
}

func (palette Palette) IsZero() bool {
	return palette == Palette{}
}

// GoLazyPalette is the terminal counterpart of the public site's brand
// palette.
var GoLazyPalette = Palette{
	Primary:   Hex("#00ADD8"),
	Secondary: Hex("#00A29C"),
	Accent:    Hex("#FDDD00"),
	Success:   Hex("#00A29C"),
	Warning:   Hex("#FDDD00"),
	Error:     Hex("#CE3262"),
}
