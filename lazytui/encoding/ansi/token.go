package ansi

import "golazy.dev/lazytui/encoding/ansi/codes"

// Token is a decoded ANSI stream item.
type Token interface {
	ansiToken()
}

// PrintToken contains printable text.
type PrintToken struct {
	Text string
}

func (PrintToken) ansiToken() {}

// ControlToken contains a single C0/C1-style control byte.
type ControlToken struct {
	Code codes.Control
}

func (ControlToken) ansiToken() {}

// EscapeToken contains an ESC sequence with a single final byte.
type EscapeToken struct {
	Final byte
	Raw   []byte
}

func (EscapeToken) ansiToken() {}

// Param is a CSI parameter. Empty parameters are preserved as nil or empty
// slices so callers can apply command-specific defaults.
type Param []int

// CSIToken contains a Control Sequence Introducer sequence.
type CSIToken struct {
	Prefix        string
	Params        []Param
	Intermediates string
	Final         byte
	Raw           []byte
}

func (CSIToken) ansiToken() {}

// OSCTerminator identifies how an OSC sequence was terminated.
type OSCTerminator uint8

const (
	OSCTerminatedByBEL OSCTerminator = iota + 1
	OSCTerminatedByST
)

// OSCToken contains an Operating System Command sequence.
type OSCToken struct {
	Command    string
	Data       string
	Terminator OSCTerminator
	Raw        []byte
}

func (OSCToken) ansiToken() {}

// WindowTitleToken reports an OSC window title command.
type WindowTitleToken struct {
	Title string
	Raw   []byte
}

func (WindowTitleToken) ansiToken() {}

// HyperlinkToken reports an OSC 8 hyperlink command. An empty URL ends the
// current hyperlink.
type HyperlinkToken struct {
	Params string
	URL    string
	Raw    []byte
}

func (HyperlinkToken) ansiToken() {}

// UnknownToken preserves a complete but unsupported or malformed sequence.
type UnknownToken struct {
	Raw    []byte
	Reason string
}

func (UnknownToken) ansiToken() {}

// WindowSizeToken reports a terminal size from an xterm-compatible response.
type WindowSizeToken struct {
	Rows int
	Cols int
	Raw  []byte
}

func (WindowSizeToken) ansiToken() {}

// MouseToken reports an SGR mouse event.
type MouseToken struct {
	Button  int
	Row     int
	Col     int
	Release bool
	Raw     []byte
}

func (MouseToken) ansiToken() {}
