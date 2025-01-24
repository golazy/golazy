package lazysupport

import (
	"bytes"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mr-tron/base58"
)

func NewValue(data any) *Value {
	if data, ok := data.([]byte); ok {
		return &Value{Data: data}
	}
	return &Value{Obj: data}
}

type Value struct {
	Data []byte
	Obj  any
}

func (v Value) IsNil() bool {
	if v.Obj == nil && v.Data == nil {
		return true
	}
	return false
}

func (v Value) IsOk() bool {
	return !v.IsNil()
}

func withBytes[T any](v Value, fn func([]byte) T) T {
	if v.Obj != nil {
		if f, ok := v.Obj.(io.Reader); ok {
			if seeker, ok := f.(io.Seeker); ok {
				defer seeker.Seek(0, 0)
			}
			data := &bytes.Buffer{}
			io.Copy(data, f)
			return fn(data.Bytes())
		}
	}
	return fn(v.Data)

}
func withReader[T any](v Value, fn func(f io.Reader) T) T {
	if v.Obj != nil {
		if f, ok := v.Obj.(io.Reader); ok {
			if seeker, ok := f.(io.Seeker); ok {
				defer seeker.Seek(0, 0)
			}
			return fn(f)
		}
	}
	return fn(bytes.NewReader(v.Data))
}

func (v Value) SHA256() Value {
	return withReader(v, func(r io.Reader) Value {
		h := sha256.New()
		io.Copy(h, r)
		return Value{Data: h.Sum(nil)}
	})
}

func (v Value) Decode64() Value {
	return withReader(v, func(r io.Reader) Value {
		reader := base64.NewDecoder(base64.RawStdEncoding, r)
		buf := bytes.Buffer{}
		_, err := io.Copy(&buf, reader)
		if err != nil {
			panic(fmt.Errorf("can't decode base64 %v: %w", v.Data, err))
		}
		return Value{Data: buf.Bytes()}
	})
}

func (v Value) Bytes() []byte {
	return withBytes(v, func(b []byte) []byte {
		return b
	})
}

func (v Value) String() string {
	return withBytes(v, func(b []byte) string {
		return string(b)
	})
}

func (v Value) Base64() Value {
	out := make([]byte, base64.RawStdEncoding.EncodedLen(len(v.Data)))
	base64.RawStdEncoding.Encode(out, v.Data)
	return Value{Data: out}
}

func (v Value) Base58() Value {
	return withBytes(v, func(data []byte) Value {

		return Value{Data: []byte(base58.FastBase58Encoding(data))}
	})

}

func (v Value) Base32() Value {
	return withBytes(v, func(data []byte) Value {
		encoded := base32.StdEncoding.EncodeToString(data)
		return Value{Data: []byte(encoded)}
	})

}

func (v Value) JSON() *Value {
	data, err := json.Marshal(v.Obj)
	if err != nil {
		panic(fmt.Errorf("can't get json from %v: %w", v.Obj, err))
	}
	return &Value{Data: data}
}

func (v Value) Int() int {
	return v.Obj.(int)
}
func (v Value) Uint() uint {
	if v.Obj == nil {
		panic("value is nil")
	}
	//panic(reflect.TypeOf(v.Obj).String())
	return v.Obj.(uint)
}
