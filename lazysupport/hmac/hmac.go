package hmac

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
)

type Encoder struct {
	block cipher.Block
}

type EncoderResult struct {
	data []byte
}

func NewEncoder(key []byte) *Encoder {
	// hash the key with sha256 so it is always 32 bytes
	hash := sha256.Sum256(key)
	key = hash[:]

	if len(key) != 32 {
		panic(fmt.Sprint("loginCodeKey must be 32 bytes: ", len(key)))
	}
	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	return &Encoder{
		block: cipherBlock,
	}
}

func (e *Encoder) EncodeString(data string) *EncoderResult {
	return e.Encode([]byte(data))
}

func (e *Encoder) JsonEncode(s any) *EncoderResult {
	data, err := json.Marshal(s)
	if err != nil {
		panic(data)
	}
	return e.Encode(data)
}

func (e *Encoder) Encode(data []byte) *EncoderResult {
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	stream := cipher.NewCFBEncrypter(e.block, iv)
	stream.XORKeyStream((ciphertext[aes.BlockSize:]), data)

	return &EncoderResult{
		data: ciphertext,
	}
}

func (e *EncoderResult) Hex() string {
	return hex.EncodeToString(e.data)
}

func (e *EncoderResult) Base64() string {
	return base64.StdEncoding.EncodeToString(e.data)
}

type Decoder struct {
	block cipher.Block
}

type DecoderResult struct {
	data []byte
}

func NewDecoder(key []byte) *Decoder {
	hash := sha256.Sum256(key)
	key = hash[:]

	cipherBlock, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	return &Decoder{
		block: cipherBlock,
	}
}

func (d *Decoder) HexDecode(hexdata string) *DecoderResult {
	data, err := hex.DecodeString(hexdata)
	if err != nil {
		panic(err)
	}
	return d.Decode(data)
}

func (d *Decoder) Base64Decode(input string) *DecoderResult {
	base64data, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		panic(err)
	}
	return d.Decode(base64data)
}

func (d *DecoderResult) UnmarshalJson(data any) {
	err := json.Unmarshal(d.data, data)
	if err != nil {
		panic(err)
	}
}

func (d *Decoder) Decode(data []byte) *DecoderResult {
	if len(data) < aes.BlockSize {
		panic(fmt.Errorf("data too short"))
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(d.block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)

	return &DecoderResult{
		data: ciphertext,
	}
}

func (d *DecoderResult) String() string {
	return string(d.data)
}
func (d *DecoderResult) Bytes() []byte {
	return d.data
}
