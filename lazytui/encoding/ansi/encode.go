package ansi

import (
	"io"
	"strconv"

	"golazy.dev/lazytui/encoding/ansi/codes"
)

// Op is an encodable terminal operation.
type Op interface {
	appendANSI([]byte) []byte
}

// Encoder writes terminal operations to an io.Writer.
type Encoder struct{}

// Encode writes op to w.
func (e *Encoder) Encode(w io.Writer, op Op) error {
	_, err := w.Write(Append(nil, op))
	return err
}

// Append appends op's ANSI byte representation to dst.
func Append(dst []byte, op Op) []byte {
	if op == nil {
		return dst
	}
	return op.appendANSI(dst)
}

type printOp string

func (p printOp) appendANSI(dst []byte) []byte {
	return append(dst, string(p)...)
}

// Print returns an operation that writes printable text.
func Print(text string) Op {
	return printOp(text)
}

type controlOp codes.Control

func (c controlOp) appendANSI(dst []byte) []byte {
	return append(dst, byte(c))
}

// Control returns an operation for a single control byte.
func Control(code codes.Control) Op {
	return controlOp(code)
}

type escapeOp byte

func (e escapeOp) appendANSI(dst []byte) []byte {
	return append(dst, byte(codes.ESC), byte(e))
}

// Escape returns an operation for a simple ESC sequence.
func Escape(final byte) Op {
	return escapeOp(final)
}

type csiOp struct {
	prefix string
	final  byte
	params []int
}

func (c csiOp) appendANSI(dst []byte) []byte {
	dst = append(dst, byte(codes.ESC), '[')
	dst = append(dst, c.prefix...)
	for i, p := range c.params {
		if i > 0 {
			dst = append(dst, ';')
		}
		dst = strconv.AppendInt(dst, int64(p), 10)
	}
	return append(dst, c.final)
}

// CSI returns an operation for a CSI sequence with numeric parameters.
func CSI(final byte, params ...int) Op {
	copied := append([]int(nil), params...)
	return csiOp{final: final, params: copied}
}

// CSIWithPrefix returns a CSI operation with a private or extension prefix.
func CSIWithPrefix(prefix string, final byte, params ...int) Op {
	copied := append([]int(nil), params...)
	return csiOp{prefix: prefix, final: final, params: copied}
}

type sequenceOp []Op

func (s sequenceOp) appendANSI(dst []byte) []byte {
	for _, op := range s {
		dst = Append(dst, op)
	}
	return dst
}

// Sequence returns an operation that writes each op in order.
func Sequence(ops ...Op) Op {
	copied := append([]Op(nil), ops...)
	return sequenceOp(copied)
}

// CursorUp moves the cursor up n rows.
func CursorUp(n int) Op {
	return CSI(codes.CUU, n)
}

// CursorDown moves the cursor down n rows.
func CursorDown(n int) Op {
	return CSI(codes.CUD, n)
}

// CursorForward moves the cursor right n columns.
func CursorForward(n int) Op {
	return CSI(codes.CUF, n)
}

// CursorBack moves the cursor left n columns.
func CursorBack(n int) Op {
	return CSI(codes.CUB, n)
}

// CursorPosition moves the cursor to row and col, both 1-based.
func CursorPosition(row, col int) Op {
	return CSI(codes.CUP, row, col)
}

// EraseDisplay erases part of the display according to mode.
func EraseDisplay(mode int) Op {
	return CSI(codes.ED, mode)
}

// EraseLine erases part of the current line according to mode.
func EraseLine(mode int) Op {
	return CSI(codes.EL, mode)
}

// RequestWindowSize asks an xterm-compatible terminal to report text-area size.
func RequestWindowSize() Op {
	return CSI('t', 18)
}

// WindowSizeReport encodes an xterm-compatible text-area size report.
func WindowSizeReport(rows, cols int) Op {
	return CSI('t', 8, rows, cols)
}

// EnableMouse enables button-event mouse tracking with SGR coordinates.
func EnableMouse() Op {
	return Sequence(
		CSIWithPrefix("?", codes.SM, codes.MouseButtonEvent),
		CSIWithPrefix("?", codes.SM, codes.MouseSGR),
	)
}

// DisableMouse disables common xterm-compatible mouse tracking modes.
func DisableMouse() Op {
	return Sequence(
		CSIWithPrefix("?", codes.RM, codes.MouseSGR),
		CSIWithPrefix("?", codes.RM, codes.MouseAnyEvent),
		CSIWithPrefix("?", codes.RM, codes.MouseButtonEvent),
		CSIWithPrefix("?", codes.RM, codes.MouseNormal),
		CSIWithPrefix("?", codes.RM, codes.MouseX10),
	)
}

type sgrOp []codes.SGR

func (s sgrOp) appendANSI(dst []byte) []byte {
	params := make([]int, 0, len(s))
	for _, part := range s {
		params = append(params, part...)
	}
	return csiOp{final: codes.SGRFinal, params: params}.appendANSI(dst)
}

// SGR returns a Select Graphic Rendition operation.
func SGR(parts ...codes.SGR) Op {
	copied := make([]codes.SGR, len(parts))
	for i, part := range parts {
		copied[i] = append(codes.SGR(nil), part...)
	}
	return sgrOp(copied)
}

type oscOp struct {
	command int
	data    string
}

func (o oscOp) appendANSI(dst []byte) []byte {
	dst = append(dst, byte(codes.ESC), ']')
	dst = strconv.AppendInt(dst, int64(o.command), 10)
	dst = append(dst, ';')
	dst = append(dst, o.data...)
	return append(dst, byte(codes.ESC), '\\')
}

// OSC returns an Operating System Command operation terminated with ST.
func OSC(command int, data string) Op {
	return oscOp{command: command, data: data}
}

// WindowTitle sets the terminal window title.
func WindowTitle(title string) Op {
	return OSC(codes.OSCSetWindowTitle, title)
}

type hyperlinkOp struct {
	params string
	url    string
}

func (h hyperlinkOp) appendANSI(dst []byte) []byte {
	dst = append(dst, byte(codes.ESC), ']')
	dst = strconv.AppendInt(dst, int64(codes.OSCHyperlink), 10)
	dst = append(dst, ';')
	dst = append(dst, h.params...)
	dst = append(dst, ';')
	dst = append(dst, h.url...)
	return append(dst, byte(codes.ESC), '\\')
}

// Hyperlink starts an OSC 8 hyperlink.
func Hyperlink(url string) Op {
	return hyperlinkOp{url: url}
}

// EndHyperlink ends the current OSC 8 hyperlink.
func EndHyperlink() Op {
	return hyperlinkOp{}
}
