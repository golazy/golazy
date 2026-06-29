package lazyconfig_test

import (
	"fmt"
	"os"

	"golazy.dev/lazyconfig"
)

func ExampleGetenv() {
	type Config struct {
		Addr  string `default:"127.0.0.1:3000"`
		Debug bool
	}

	os.Setenv("DEBUG", "true")
	defer os.Unsetenv("DEBUG")

	config, err := lazyconfig.Getenv[Config]()
	if err != nil {
		panic(err)
	}

	fmt.Println(config.Addr)
	fmt.Println(config.Debug)

	// Output:
	// 127.0.0.1:3000
	// true
}
