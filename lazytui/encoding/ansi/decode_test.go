package ansi

import (
	"reflect"
	"testing"

	"golazy.dev/lazytui/encoding/ansi/codes"
)

func TestDecoderPrintableText(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "hello")

	want := []Token{PrintToken{Text: "hello"}}
	requireTokens(t, tokens, want)
}

func TestDecoderUTF8AcrossChunks(t *testing.T) {
	var decoder Decoder

	first := decodeAll(t, &decoder, "pi \xcf")
	requireTokens(t, first, []Token{PrintToken{Text: "pi "}})

	second := decodeAll(t, &decoder, "\x80")
	requireTokens(t, second, []Token{PrintToken{Text: "π"}})
}

func TestDecoderControlFlushesText(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "a\r\nb")

	want := []Token{
		PrintToken{Text: "a"},
		ControlToken{Code: codes.CR},
		ControlToken{Code: codes.LF},
		PrintToken{Text: "b"},
	}
	requireTokens(t, tokens, want)
}

func TestDecoderEscapeToken(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b7")

	want := []Token{EscapeToken{Final: '7', Raw: []byte("\x1b7")}}
	requireTokens(t, tokens, want)
}

func TestDecoderCSI(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b[12;4H")

	want := []Token{CSIToken{
		Params: []Param{{12}, {4}},
		Final:  codes.CUP,
		Raw:    []byte("\x1b[12;4H"),
	}}
	requireTokens(t, tokens, want)
}

func TestDecoderCSIPrefixAndEmptyParams(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b[?25l\x1b[;H")

	want := []Token{
		CSIToken{
			Prefix: "?",
			Params: []Param{{25}},
			Final:  'l',
			Raw:    []byte("\x1b[?25l"),
		},
		CSIToken{
			Params: []Param{nil, nil},
			Final:  codes.CUP,
			Raw:    []byte("\x1b[;H"),
		},
	}
	requireTokens(t, tokens, want)
}

func TestDecoderCSIWithSubparameters(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b[38:2:1:2:3m")

	want := []Token{CSIToken{
		Params: []Param{{38, 2, 1, 2, 3}},
		Final:  codes.SGRFinal,
		Raw:    []byte("\x1b[38:2:1:2:3m"),
	}}
	requireTokens(t, tokens, want)
}

func TestDecoderWindowSizeToken(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b[8;24;80t")

	want := []Token{WindowSizeToken{Rows: 24, Cols: 80, Raw: []byte("\x1b[8;24;80t")}}
	requireTokens(t, tokens, want)
}

func TestDecoderMouseToken(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b[<0;12;7M\x1b[<0;12;7m")

	want := []Token{
		MouseToken{Button: 0, Col: 12, Row: 7, Raw: []byte("\x1b[<0;12;7M")},
		MouseToken{Button: 0, Col: 12, Row: 7, Release: true, Raw: []byte("\x1b[<0;12;7m")},
	}
	requireTokens(t, tokens, want)
}

func TestDecoderSplitCSI(t *testing.T) {
	var decoder Decoder

	first := decodeAll(t, &decoder, "\x1b[12")
	requireTokens(t, first, nil)

	second := decodeAll(t, &decoder, ";4H")
	want := []Token{CSIToken{
		Params: []Param{{12}, {4}},
		Final:  codes.CUP,
		Raw:    []byte("\x1b[12;4H"),
	}}
	requireTokens(t, second, want)
}

func TestDecoderOSCWithBEL(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b]0;Lazy\a")

	want := []Token{WindowTitleToken{Title: "Lazy", Raw: []byte("\x1b]0;Lazy\a")}}
	requireTokens(t, tokens, want)
}

func TestDecoderOSCWithSTAcrossChunks(t *testing.T) {
	var decoder Decoder

	first := decodeAll(t, &decoder, "\x1b]2;La")
	requireTokens(t, first, nil)

	second := decodeAll(t, &decoder, "zy\x1b\\")
	want := []Token{WindowTitleToken{Title: "Lazy", Raw: []byte("\x1b]2;Lazy\x1b\\")}}
	requireTokens(t, second, want)
}

func TestDecoderOSCHyperlink(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b]8;id=lazy;https://golazy.dev\x1b\\link\x1b]8;;\x1b\\")

	want := []Token{
		HyperlinkToken{Params: "id=lazy", URL: "https://golazy.dev", Raw: []byte("\x1b]8;id=lazy;https://golazy.dev\x1b\\")},
		PrintToken{Text: "link"},
		HyperlinkToken{Raw: []byte("\x1b]8;;\x1b\\")},
	}
	requireTokens(t, tokens, want)
}

func TestDecoderOSCGeneric(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1b]1337;payload\x1b\\")

	want := []Token{OSCToken{
		Command:    "1337",
		Data:       "payload",
		Terminator: OSCTerminatedByST,
		Raw:        []byte("\x1b]1337;payload\x1b\\"),
	}}
	requireTokens(t, tokens, want)
}

func TestDecoderUnsupportedStringSequence(t *testing.T) {
	var decoder Decoder
	tokens := decodeAll(t, &decoder, "\x1bPpayload\x1b\\")

	want := []Token{UnknownToken{
		Raw:    []byte("\x1bPpayload\x1b\\"),
		Reason: "unsupported string sequence",
	}}
	requireTokens(t, tokens, want)
}

func TestDecoderResetDropsIncompleteSequence(t *testing.T) {
	var decoder Decoder

	requireTokens(t, decodeAll(t, &decoder, "\x1b[12"), nil)
	decoder.Reset()
	requireTokens(t, decodeAll(t, &decoder, "x"), []Token{PrintToken{Text: "x"}})
}

func FuzzDecoderDoesNotPanic(f *testing.F) {
	seeds := []string{
		"hello",
		"\x1b[31mred\x1b[0m",
		"\x1b]0;title\a",
		"\x1b]8;;https://golazy.dev\x1b\\link\x1b]8;;\x1b\\",
		"\x1b[?1000;1006h",
		"\xff\xfe\xfd",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		var decoder Decoder
		if _, err := decoder.Decode([]byte(input)); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
	})
}

func decodeAll(t *testing.T, decoder *Decoder, input string) []Token {
	t.Helper()
	tokens, err := decoder.Decode([]byte(input))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	return tokens
}

func requireTokens(t *testing.T, got []Token, want []Token) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens mismatch\n got: %#v\nwant: %#v", got, want)
	}
}
