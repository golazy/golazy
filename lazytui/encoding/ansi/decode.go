package ansi

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"golazy.dev/lazytui/encoding/ansi/codes"
)

type decoderState uint8

const (
	stateGround decoderState = iota
	stateEscape
	stateCSI
	stateOSC
	stateOSCEscape
	stateString
	stateStringEscape
)

// Decoder incrementally decodes ANSI and VT terminal streams.
type Decoder struct {
	state decoderState
	text  []byte
	seq   []byte
}

// Reset clears the decoder state and drops buffered incomplete input.
func (d *Decoder) Reset() {
	d.state = stateGround
	d.text = nil
	d.seq = nil
}

// Decode consumes data and returns any complete tokens found.
func (d *Decoder) Decode(data []byte) ([]Token, error) {
	var tokens []Token
	for _, b := range data {
		switch d.state {
		case stateGround:
			d.consumeGround(b, &tokens)
		case stateEscape:
			d.consumeEscape(b, &tokens)
		case stateCSI:
			d.consumeCSI(b, &tokens)
		case stateOSC:
			d.consumeOSC(b, &tokens)
		case stateOSCEscape:
			d.consumeOSCEscape(b, &tokens)
		case stateString:
			d.consumeString(b, &tokens)
		case stateStringEscape:
			d.consumeStringEscape(b, &tokens)
		}
	}
	d.flushText(&tokens, false)
	return tokens, nil
}

func (d *Decoder) consumeGround(b byte, tokens *[]Token) {
	switch {
	case b == byte(codes.ESC):
		d.flushText(tokens, true)
		d.state = stateEscape
		d.seq = append(d.seq[:0], b)
	case b < 0x20 || b == byte(codes.DEL):
		d.flushText(tokens, true)
		*tokens = append(*tokens, ControlToken{Code: codes.Control(b)})
	default:
		d.text = append(d.text, b)
	}
}

func (d *Decoder) consumeEscape(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	switch b {
	case '[':
		d.state = stateCSI
	case ']':
		d.state = stateOSC
	case 'P', '^', '_', 'X':
		d.state = stateString
	default:
		*tokens = append(*tokens, EscapeToken{Final: b, Raw: cloneBytes(d.seq)})
		d.seq = nil
		d.state = stateGround
	}
}

func (d *Decoder) consumeCSI(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	if b >= 0x40 && b <= 0x7e {
		token, ok := parseCSI(d.seq)
		if ok {
			*tokens = append(*tokens, specializeCSI(token))
		} else {
			*tokens = append(*tokens, UnknownToken{Raw: cloneBytes(d.seq), Reason: "malformed CSI"})
		}
		d.seq = nil
		d.state = stateGround
	}
}

func specializeCSI(token CSIToken) Token {
	if token.Final == 't' && len(token.Params) == 3 && paramValue(token.Params[0], -1) == 8 {
		rows := paramValue(token.Params[1], 0)
		cols := paramValue(token.Params[2], 0)
		if rows > 0 && cols > 0 {
			return WindowSizeToken{Rows: rows, Cols: cols, Raw: cloneBytes(token.Raw)}
		}
	}
	if token.Prefix == "<" && len(token.Params) == 3 && (token.Final == 'M' || token.Final == 'm') {
		button := paramValue(token.Params[0], -1)
		col := paramValue(token.Params[1], 0)
		row := paramValue(token.Params[2], 0)
		if button >= 0 && row > 0 && col > 0 {
			return MouseToken{
				Button:  button,
				Row:     row,
				Col:     col,
				Release: token.Final == 'm',
				Raw:     cloneBytes(token.Raw),
			}
		}
	}
	return token
}

func paramValue(param Param, fallback int) int {
	if len(param) == 0 {
		return fallback
	}
	return param[0]
}

func (d *Decoder) consumeOSC(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	switch b {
	case byte(codes.BEL):
		*tokens = append(*tokens, specializeOSC(parseOSC(d.seq, OSCTerminatedByBEL)))
		d.seq = nil
		d.state = stateGround
	case byte(codes.ESC):
		d.state = stateOSCEscape
	}
}

func (d *Decoder) consumeOSCEscape(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	if b == '\\' {
		*tokens = append(*tokens, specializeOSC(parseOSC(d.seq, OSCTerminatedByST)))
		d.seq = nil
		d.state = stateGround
		return
	}
	d.state = stateOSC
}

func (d *Decoder) consumeString(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	switch b {
	case byte(codes.BEL):
		*tokens = append(*tokens, UnknownToken{Raw: cloneBytes(d.seq), Reason: "unsupported string sequence"})
		d.seq = nil
		d.state = stateGround
	case byte(codes.ESC):
		d.state = stateStringEscape
	}
}

func (d *Decoder) consumeStringEscape(b byte, tokens *[]Token) {
	d.seq = append(d.seq, b)
	if b == '\\' {
		*tokens = append(*tokens, UnknownToken{Raw: cloneBytes(d.seq), Reason: "unsupported string sequence"})
		d.seq = nil
		d.state = stateGround
		return
	}
	d.state = stateString
}

func (d *Decoder) flushText(tokens *[]Token, force bool) {
	if len(d.text) == 0 {
		return
	}
	n := validTextPrefix(d.text)
	if force {
		n = len(d.text)
	}
	if n > 0 {
		*tokens = append(*tokens, PrintToken{Text: string(d.text[:n])})
	}
	if n == len(d.text) {
		d.text = nil
		return
	}
	copy(d.text, d.text[n:])
	d.text = d.text[:len(d.text)-n]
}

func validTextPrefix(data []byte) int {
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 && !utf8.FullRune(data[i:]) {
			return i
		}
		i += size
	}
	return len(data)
}

func parseCSI(raw []byte) (CSIToken, bool) {
	if len(raw) < 3 || raw[0] != byte(codes.ESC) || raw[1] != '[' {
		return CSIToken{}, false
	}
	final := raw[len(raw)-1]
	body := string(raw[2 : len(raw)-1])
	paramEnd := 0
	for paramEnd < len(body) && body[paramEnd] >= 0x30 && body[paramEnd] <= 0x3f {
		paramEnd++
	}

	paramText := body[:paramEnd]
	intermediates := body[paramEnd:]
	for i := 0; i < len(intermediates); i++ {
		if intermediates[i] < 0x20 || intermediates[i] > 0x2f {
			return CSIToken{}, false
		}
	}

	prefix := ""
	if paramText != "" && strings.ContainsRune("<=>?", rune(paramText[0])) {
		prefix = paramText[:1]
		paramText = paramText[1:]
	}

	params, ok := parseParams(paramText)
	if !ok {
		return CSIToken{}, false
	}

	return CSIToken{
		Prefix:        prefix,
		Params:        params,
		Intermediates: intermediates,
		Final:         final,
		Raw:           cloneBytes(raw),
	}, true
}

func parseParams(text string) ([]Param, bool) {
	if text == "" {
		return nil, true
	}
	parts := strings.Split(text, ";")
	params := make([]Param, len(parts))
	for i, part := range parts {
		if part == "" {
			continue
		}
		subparts := strings.Split(part, ":")
		params[i] = make(Param, len(subparts))
		for j, subpart := range subparts {
			if subpart == "" {
				continue
			}
			value, err := strconv.Atoi(subpart)
			if err != nil {
				return nil, false
			}
			params[i][j] = value
		}
	}
	return params, true
}

func parseOSC(raw []byte, terminator OSCTerminator) OSCToken {
	bodyEnd := len(raw) - 1
	if terminator == OSCTerminatedByST {
		bodyEnd = len(raw) - 2
	}
	body := string(raw[2:bodyEnd])
	command, data, _ := strings.Cut(body, ";")
	return OSCToken{
		Command:    command,
		Data:       data,
		Terminator: terminator,
		Raw:        cloneBytes(raw),
	}
}

func specializeOSC(token OSCToken) Token {
	switch token.Command {
	case strconv.Itoa(codes.OSCSetIconAndTitle), strconv.Itoa(codes.OSCSetWindowTitle):
		return WindowTitleToken{Title: token.Data, Raw: cloneBytes(token.Raw)}
	case strconv.Itoa(codes.OSCHyperlink):
		params, url, _ := strings.Cut(token.Data, ";")
		return HyperlinkToken{Params: params, URL: url, Raw: cloneBytes(token.Raw)}
	default:
		return token
	}
}

func cloneBytes(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	return append([]byte(nil), data...)
}
