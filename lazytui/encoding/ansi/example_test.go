package ansi

import (
	"bytes"
	"fmt"

	"golazy.dev/lazytui/encoding/ansi/codes"
)

func ExampleEncoder() {
	var buf bytes.Buffer
	encoder := &Encoder{}

	_ = encoder.Encode(&buf, SGR(codes.Bold, codes.FgRGB(1, 2, 3)))
	_ = encoder.Encode(&buf, Print("Lazy"))
	_ = encoder.Encode(&buf, SGR(codes.Reset))

	fmt.Printf("%q\n", buf.String())
	// Output: "\x1b[1;38;2;1;2;3mLazy\x1b[0m"
}

func ExampleDecoder() {
	var decoder Decoder
	tokens, _ := decoder.Decode([]byte("\x1b[2JHi"))

	for _, token := range tokens {
		fmt.Printf("%T\n", token)
	}
	// Output:
	// ansi.CSIToken
	// ansi.PrintToken
}
