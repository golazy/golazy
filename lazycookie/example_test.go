package lazycookie_test

import (
	"fmt"

	"golazy.dev/lazycookie"
)

func Example() {
	codec := lazycookie.New(
		[]byte("32-byte-authentication-secret!!"),
		nil,
	)

	encoded, err := codec.Encode("preferences", map[string]string{
		"theme": "dark",
	})
	if err != nil {
		panic(err)
	}

	var decoded map[string]string
	if err := codec.Decode("preferences", encoded, &decoded); err != nil {
		panic(err)
	}

	fmt.Println(decoded["theme"])
	// Output: dark
}
