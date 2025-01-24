package hmac

import "testing"

const key = "This is my key"

func TestHmac(t *testing.T) {

	encoder := NewEncoder([]byte(key))
	hexdata := encoder.EncodeString("hola").Hex()
	t.Log(hexdata)

	decoder := NewDecoder([]byte(key))
	msg := decoder.HexDecode(hexdata).String()
	if msg != "hola" {
		t.Error("Expected hola, got ", msg)
	}

}

func TestJsonEncode(t *testing.T) {

	data := map[string]string{"hola": "mundo"}
	encoder := NewEncoder([]byte(key))
	hexdata := encoder.JsonEncode(data).Hex()
	t.Log(hexdata)

	var out map[string]string
	decoder := NewDecoder([]byte(key))
	decoder.HexDecode(hexdata).UnmarshalJson(&out)
	if out["hola"] != "mundo" {
		t.Fatal(out)
	}

}

func TestBase64(t *testing.T) {

	encoder := NewEncoder([]byte(key))
	encoded := encoder.EncodeString("hola").Base64()
	t.Log(encoded)

	decoder := NewDecoder([]byte(key))
	msg := decoder.Base64Decode(encoded).String()
	if msg != "hola" {
		t.Error("Expected hola, got ", msg)
	}

}
