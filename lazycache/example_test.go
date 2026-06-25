package lazycache_test

import (
	"fmt"

	"golazy.dev/lazycache"
	"golazy.dev/lazycache/inmemorycache"
)

func Example() {
	backend, err := inmemorycache.New(inmemorycache.Options{})
	if err != nil {
		panic(err)
	}
	cache, err := lazycache.New(lazycache.Options{Backend: backend})
	if err != nil {
		panic(err)
	}

	_ = lazycache.Set(cache, "Ada", "user", 1)
	name, _ := lazycache.Get[string](cache, "user", 1)
	fmt.Println(name)
	// Output: Ada
}
