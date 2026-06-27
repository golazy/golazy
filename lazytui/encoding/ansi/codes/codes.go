// Package codes defines ANSI, ECMA-48, and common terminal extension codes.
//
// The package intentionally contains constants and small value helpers only.
// Stateful parsing and rendering belong in golazy.dev/lazytui/encoding/ansi.
package codes

// Control is a C0/C1-style control byte.
type Control byte

const (
	NUL Control = 0x00
	SOH Control = 0x01
	STX Control = 0x02
	ETX Control = 0x03
	EOT Control = 0x04
	ENQ Control = 0x05
	ACK Control = 0x06
	BEL Control = 0x07
	BS  Control = 0x08
	HT  Control = 0x09
	LF  Control = 0x0a
	VT  Control = 0x0b
	FF  Control = 0x0c
	CR  Control = 0x0d
	SO  Control = 0x0e
	SI  Control = 0x0f
	DLE Control = 0x10
	DC1 Control = 0x11
	DC2 Control = 0x12
	DC3 Control = 0x13
	DC4 Control = 0x14
	NAK Control = 0x15
	SYN Control = 0x16
	ETB Control = 0x17
	CAN Control = 0x18
	EM  Control = 0x19
	SUB Control = 0x1a
	ESC Control = 0x1b
	FS  Control = 0x1c
	GS  Control = 0x1d
	RS  Control = 0x1e
	US  Control = 0x1f
	DEL Control = 0x7f
)

// CSI final bytes for common control sequence functions.
const (
	CUU      byte = 'A'
	CUD      byte = 'B'
	CUF      byte = 'C'
	CUB      byte = 'D'
	CNL      byte = 'E'
	CPL      byte = 'F'
	CHA      byte = 'G'
	CUP      byte = 'H'
	ED       byte = 'J'
	EL       byte = 'K'
	SU       byte = 'S'
	SD       byte = 'T'
	HVP      byte = 'f'
	SM       byte = 'h'
	RM       byte = 'l'
	SGRFinal byte = 'm'
)

// Private mode numbers for common DEC/xterm mouse tracking modes.
const (
	MouseX10         = 9
	MouseNormal      = 1000
	MouseButtonEvent = 1002
	MouseAnyEvent    = 1003
	MouseSGR         = 1006
)

// OSC command numbers used by xterm-compatible terminals.
const (
	OSCSetIconAndTitle = 0
	OSCSetIconName     = 1
	OSCSetWindowTitle  = 2
	OSCHyperlink       = 8
)

// SGR is a Select Graphic Rendition parameter sequence.
type SGR []int

var (
	Reset           = SGR{0}
	Bold            = SGR{1}
	Faint           = SGR{2}
	Italic          = SGR{3}
	Underline       = SGR{4}
	Blink           = SGR{5}
	Inverse         = SGR{7}
	Conceal         = SGR{8}
	CrossedOut      = SGR{9}
	NormalIntensity = SGR{22}
	NoItalic        = SGR{23}
	NoUnderline     = SGR{24}
	NoBlink         = SGR{25}
	Positive        = SGR{27}
	Reveal          = SGR{28}
	NotCrossedOut   = SGR{29}
	DefaultFg       = SGR{39}
	DefaultBg       = SGR{49}
	Framed          = SGR{51}
	Encircled       = SGR{52}
	Overline        = SGR{53}
	NoFrameCircle   = SGR{54}
	NoOverline      = SGR{55}
)

// Fg256 selects an indexed foreground color.
func Fg256(index uint8) SGR {
	return SGR{38, 5, int(index)}
}

// Bg256 selects an indexed background color.
func Bg256(index uint8) SGR {
	return SGR{48, 5, int(index)}
}

// FgRGB selects a truecolor foreground color.
func FgRGB(r, g, b uint8) SGR {
	return SGR{38, 2, int(r), int(g), int(b)}
}

// BgRGB selects a truecolor background color.
func BgRGB(r, g, b uint8) SGR {
	return SGR{48, 2, int(r), int(g), int(b)}
}
